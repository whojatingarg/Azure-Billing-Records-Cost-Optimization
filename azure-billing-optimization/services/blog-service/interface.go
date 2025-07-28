INTERFACE BlobService {
    METHODS:
        UploadBlob(ctx Context, containerName String, blobName String, data []Byte) -> UploadResult
        DownloadBlob(ctx Context, containerName String, blobName String) -> DownloadResult
        SetBlobTier(ctx Context, blobName String, accessTier AccessTier) -> TierResult
        GetBlobProperties(ctx Context, containerName String, blobName String) -> BlobProperties
        ListBlobsByPrefix(ctx Context, containerName String, prefix String) -> []BlobInfo
}