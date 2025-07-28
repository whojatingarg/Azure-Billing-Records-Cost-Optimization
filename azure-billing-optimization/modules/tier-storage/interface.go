INTERFACE TierStorage {
    METHODS:
        WriteToTier(ctx Context, tier DataTier, record BillingRecord) -> WriteResult
        ReadFromTier(ctx Context, tier DataTier, recordID String) -> ReadResult
        DeleteFromTier(ctx Context, tier DataTier, recordID String) -> DeleteResult
        GetTierMetrics(ctx Context, tier DataTier) -> TierMetrics
}

MODULE TierStorageModule IMPLEMENTS TierStorage {
    DEPENDENCIES:
        - CosmosClient
        - BlobClient
        - CacheClient
        - EncryptionService
        - CompressionService
        
    CONFIGURATION:
        - TierConfigurations: Map<DataTier, TierConfig>
        - EncryptionEnabled: Boolean
        - CompressionThreshold: Integer
        - RetryPolicy: RetryPolicy
        
    METHOD WriteToTier(ctx, tier, record) {
        START_TIMER(writeTimer)
        
        TRY {
            // Step 1: Pre-process data based on tier requirements
            processedData = PreProcessData(record, tier)
            
            // Step 2: Apply encryption if enabled
            IF ENCRYPTION_ENABLED {
                processedData = EncryptionService.Encrypt(processedData)
            }
            
            // Step 3: Apply compression for large records
            IF processedData.Size > COMPRESSION_THRESHOLD {
                processedData = CompressionService.Compress(processedData)
            }
            
            // Step 4: Execute tier-specific write operation
            SWITCH tier {
                CASE HOT_TIER:
                    result = WriteToCosmosDB(ctx, processedData)
                    
                CASE WARM_TIER:
                    result = WriteToBlobStorage(ctx, processedData, AccessTier.HOT)
                    
                CASE COLD_TIER:
                    result = WriteToBlobStorage(ctx, processedData, AccessTier.ARCHIVE)
            }
            
            // Step 5: Validate write success
            IF result.Success {
                RECORD_METRIC("write_success", tier)
                RECORD_METRIC("write_latency", tier, STOP_TIMER(writeTimer))
                RECORD_METRIC("data_size_written", tier, processedData.Size)
            }
            
            RETURN result
            
        } CATCH Exception e {
            RECORD_METRIC("write_failure", tier)
            THROW e
        }
    }
    
    METHOD ReadFromTier(ctx, tier, recordID) {
        START_TIMER(readTimer)
        
        TRY {
            // Step 1: Execute tier-specific read operation
            SWITCH tier {
                CASE HOT_TIER:
                    rawData = ReadFromCosmosDB(ctx, recordID)
                    
                CASE WARM_TIER:
                    rawData = ReadFromBlobStorage(ctx, recordID, AccessTier.HOT)
                    
                CASE COLD_TIER:
                    // Handle rehydration for archived blobs
                    IF IsArchived(recordID) {
                        RETURN InitiateRehydration(ctx, recordID)
                    } ELSE {
                        rawData = ReadFromBlobStorage(ctx, recordID, AccessTier.COOL)
                    }
            }
            
            // Step 2: Post-process data
            processedData = PostProcessData(rawData, tier)
            
            // Step 3: Decrypt if encrypted
            IF IsEncrypted(processedData) {
                processedData = EncryptionService.Decrypt(processedData)
            }
            
            // Step 4: Decompress if compressed
            IF IsCompressed(processedData) {
                processedData = CompressionService.Decompress(processedData)
            }
            
            RECORD_METRIC("read_success", tier)
            RECORD_METRIC("read_latency", tier, STOP_TIMER(readTimer))
            
            RETURN ReadResult{
                Success: TRUE,
                Data: processedData,
                Tier: tier
            }
            
        } CATCH Exception e {
            RECORD_METRIC("read_failure", tier)
            THROW e
        }
    }
    
    PRIVATE METHOD InitiateRehydration(ctx, recordID) {
        // Step 1: Change blob access tier to Hot
        rehydrationRequest = BlobRehydrationRequest{
            RecordID: recordID,
            TargetTier: AccessTier.HOT,
            Priority: RehydrationPriority.STANDARD
        }
        
        BlobClient.SetBlobTier(ctx, recordID, rehydrationRequest)
        
        // Step 2: Return async response with location
        RETURN ReadResult{
            Success: TRUE,
            IsAsync: TRUE,
            Location: "/api/rehydration-status/" + recordID,
            EstimatedWaitTime: GetRehydrationEstimate()
        }
    }
}