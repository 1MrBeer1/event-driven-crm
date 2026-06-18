# Compose profiles

The root `docker-compose.yml` contains the runnable local stack.

Useful commands:

- Infrastructure only: `docker compose up -d postgres redis kafka kafka-ui`
- Full CRM stack: `docker compose up --build`
- Migrations only: `docker compose run --rm migrate`
