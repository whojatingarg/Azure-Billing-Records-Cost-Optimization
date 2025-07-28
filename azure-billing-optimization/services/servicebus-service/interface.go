INTERFACE ServiceBusService {
    METHODS:
        EnqueueMessage(queueName String, message Message) -> EnqueueResult
        ReceiveMessages(queueName String, maxMessages Integer, timeout Duration) -> []Message
        CompleteMessage(message Message) -> Void
        DeadLetterMessage(message Message, reason String) -> Void
        GetQueueMetrics(queueName String) -> QueueMetrics
}