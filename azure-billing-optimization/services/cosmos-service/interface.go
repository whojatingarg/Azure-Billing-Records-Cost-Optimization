INTERFACE CosmosService {
    METHODS:
        GetRecord(ctx Context, recordID String) -> BillingRecord
        CreateRecord(ctx Context, record BillingRecord) -> CreateResult
        QueryMigrationCandidates(cutoffDate Time, batchSize Integer) -> []BillingRecord
        MarkAsMigrated(ctx Context, recordID String, targetTier DataTier) -> UpdateResult
        BulkDelete(ctx Context, recordIDs []String) -> BulkDeleteResult
}