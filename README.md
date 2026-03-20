# pokemon-web (local)

Antes podías jugar en este PC porque el backend se estaba ejecutando con una base de datos configurada. Hoy falla si no existe `DATABASE_URL`.

## Levantar en local (sin instalar Postgres)

Requisitos:
- Docker + Docker Compose

Comando:
```bash
./scripts/dev.sh
```

Esto:
- levanta Postgres con `docker compose`
- exporta `DATABASE_URL=postgres://pokemon:pokemon@localhost:5432/pokemon_web?sslmode=disable`
- arranca `./backend/pokemon-web` en `PORT=3030`

Abrí:
- `http://localhost:3030`

## Variables útiles

- `DATABASE_URL` (obligatoria)
- `JWT_SECRET` (opcional)
- `PORT` (default `3030`)
- `ROMS_DIR` y `FRONTEND_DIR` (opcionales)

