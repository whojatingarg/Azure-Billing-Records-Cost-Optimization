INTERFACE DataRouter {
    METHODS:
        RouteRequest(ctx Context, request BillingRequest) -> TierResponse
        DetermineOptimalTier(recordAge Duration, accessPattern AccessPattern) -> DataTier
        ExecuteCircuitBreaker(tier DataTier, operation Operation) -> Result
        CacheResult(key String, data BillingRecord, ttl Duration) -> Boolean
}

MODULE DataRouterModule IMPLEMENTS DataRouter {
    DEPENDENCIES:
        - CosmosService
        - BlobService  
        - CacheService
        - MetricsCollector
        - CircuitBreaker
        
    CONFIGURATION:
        - TierThresholds: Map<DataTier, Duration>
        - CircuitBreakerConfig: CircuitBreakerSettings
        - CacheConfig: CacheSettings
        - RetryPolicy: RetrySettings
        
    METHOD RouteRequest(ctx, request) {
        START_TIMER(routingTimer)
        
        // Step 1: Check cache first
        IF cachedResult = CacheService.Get(request.RecordID) {
            RECORD_METRIC("cache_hit", request.RecordID)
            RETURN cachedResult WITH tier=CACHE
        }
        
        // Step 2: Determine target tier based on age and access pattern
        targetTier = DetermineOptimalTier(request.RecordAge, request.AccessPattern)
        
        // Step 3: Execute with circuit breaker protection
        SWITCH targetTier {
            CASE HOT_TIER:
                result = ExecuteCircuitBreaker(HOT_TIER, () => {
                    RETURN CosmosService.GetRecord(ctx, request.RecordID)
                })
                
            CASE WARM_TIER:
                result = ExecuteCircuitBreaker(WARM_TIER, () => {
                    RETURN BlobService.GetFromWarmTier(ctx, request.RecordID)
                })
                
            CASE COLD_TIER:
                result = ExecuteCircuitBreaker(COLD_TIER, () => {
                    RETURN BlobService.InitiateRehydration(ctx, request.RecordID)
                })
        }
        
        // Step 4: Cache successful results
        IF result.Success AND targetTier != COLD_TIER {
            CacheService.Set(request.RecordID, result.Data, GetCacheTTL(targetTier))
        }
        
        RECORD_METRIC("routing_latency", STOP_TIMER(routingTimer))
        RECORD_METRIC("tier_access", targetTier)
        
        RETURN result
    }
    
    METHOD DetermineOptimalTier(recordAge, accessPattern) {
        IF recordAge < HOT_TIER_THRESHOLD {
            RETURN HOT_TIER
        }
        
        IF recordAge < WARM_TIER_THRESHOLD AND accessPattern.Frequency > WARM_THRESHOLD {
            RETURN WARM_TIER
        }
        
        RETURN COLD_TIER
    }
    
    METHOD ExecuteCircuitBreaker(tier, operation) {
        circuitBreaker = GetCircuitBreakerForTier(tier)
        
        TRY {
            result = circuitBreaker.Execute(operation)
            RECORD_METRIC("circuit_breaker_success", tier)
            RETURN result
        } CATCH CircuitBreakerOpenException {
            RECORD_METRIC("circuit_breaker_open", tier)
            RETURN FallbackResponse(tier)
        } CATCH Exception e {
            RECORD_METRIC("circuit_breaker_failure", tier)
            THROW e
        }
    }
}