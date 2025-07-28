INTERFACE SecurityService {
    METHODS:
        ValidateApiKey(apiKey String) -> ValidationResult
        EncryptSensitiveData(data BillingRecord) -> EncryptedData
        AuditDataAccess(operation String, recordID String, userContext UserContext) -> Void
        CheckDataRetentionCompliance(record BillingRecord) -> ComplianceResult
}

MODULE SecurityModule IMPLEMENTS SecurityService {
    DEPENDENCIES:
        - KeyVaultClient
        - AuditLogger
        - ComplianceChecker
        - EncryptionProvider
        
    CONFIGURATION:
        - EncryptionKeyRotationPolicy: RotationPolicy
        - DataRetentionPolicies: Map<DataType, RetentionPolicy>
        - AuditConfig: AuditConfiguration
        
    METHOD ValidateApiKey(apiKey) {
        TRY {
            // Step 1: Extract key metadata
            keyMetadata = ExtractKeyMetadata(apiKey)
            
            // Step 2: Validate key format and signature
            IF NOT IsValidKeyFormat(apiKey) {
                RETURN ValidationResult{Valid: FALSE, Reason: "Invalid key format"}
            }
            
            // Step 3: Check key against Key Vault
            keyDetails = KeyVaultClient.GetKeyDetails(keyMetadata.KeyID)
            
            // Step 4: Validate key expiration and revocation status
            IF keyDetails.IsExpired() OR keyDetails.IsRevoked() {
                RETURN ValidationResult{Valid: FALSE, Reason: "Key expired or revoked"}
            }
            
            // Step 5: Check rate limiting
            IF NOT CheckRateLimit(keyMetadata.ClientID) {
                RETURN ValidationResult{Valid: FALSE, Reason: "Rate limit exceeded"}
            }
            
            RETURN ValidationResult{
                Valid: TRUE,
                ClientID: keyMetadata.ClientID,
                Permissions: keyDetails.Permissions
            }
            
        } CATCH Exception e {
            AuditLogger.LogSecurityEvent("API_KEY_VALIDATION_FAILED", apiKey, e)
            RETURN ValidationResult{Valid: FALSE, Reason: "Validation error"}
        }
    }
    
    METHOD EncryptSensitiveData(data) {
        // Step 1: Identify sensitive fields
        sensitiveFields = IdentifySensitiveFields(data)
        
        // Step 2: Get current encryption key
        encryptionKey = KeyVaultClient.GetCurrentEncryptionKey()
        
        // Step 3: Encrypt sensitive fields
        encryptedData = data.Clone()
        FOR EACH field IN sensitiveFields {
            encryptedData.SetField(field.Name, EncryptionProvider.Encrypt(
                field.Value, 
                encryptionKey,
                EncryptionAlgorithm.AES256_GCM
            ))
        }
        
        // Step 4: Add encryption metadata
        encryptedData.EncryptionMetadata = EncryptionMetadata{
            KeyVersion: encryptionKey.Version,
            Algorithm: EncryptionAlgorithm.AES256_GCM,
            EncryptedFields: sensitiveFields.Names(),
            Timestamp: NOW()
        }
        
        RETURN encryptedData
    }
    
    METHOD AuditDataAccess(operation, recordID, userContext) {
        auditEvent = DataAccessAuditEvent{
            Operation: operation,
            RecordID: recordID,
            UserID: userContext.UserID,
            ClientIP: userContext.ClientIP,
            Timestamp: NOW(),
            RequestID: userContext.RequestID,
            DataClassification: GetDataClassification(recordID)
        }
        
        // Log to audit system
        AuditLogger.LogDataAccess(auditEvent)
        
        // Send to SIEM if high-risk operation
        IF IsHighRiskOperation(operation) {
            SIEMConnector.SendSecurityEvent(auditEvent)
        }
        
        // Check for suspicious access patterns
        CheckForAnomalousAccess(userContext, operation)
    }
}