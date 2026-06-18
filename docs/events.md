# Event Contracts

Kafka topics:

- `lead.created`
- `customer.created`
- `notification.created`
- `lead.created.dlq`
- `customer.created.dlq`

Redis Pub/Sub channel:

- `crm.notifications`

The service consumers use explicit commits. Processing failures are retried three times and then moved to a DLQ topic with the source payload and error metadata.

