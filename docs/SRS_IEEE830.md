# Software Requirements Specification (SRS) - IEEE 830 (Baseline)

Proyecto: `pokemon-web`  
Fecha: 2026-02-16  
Estado: Borrador (derivado del repo actual, sin suposiciones adicionales)

## 1. Introduccion

### 1.1 Proposito
Este documento define los requerimientos del sistema `pokemon-web` siguiendo la estructura de IEEE 830, con un nivel conservador basado en lo observable en el repositorio (frontend estatico + backend Go + DB PostgreSQL).

### 1.2 Alcance
`pokemon-web` provee una aplicacion web para:
- Autenticacion (registro e inicio de sesion).
- Gestion de ROMs (listar, subir) y descarga publica para el emulador.
- Gestion de partidas guardadas (crear/listar/actualizar/borrar) asociadas al usuario autenticado.
- Gestion de datos de Nuzlocke asociados a una partida guardada.

### 1.3 Definiciones, acronimos y abreviaturas
- API: Interfaz HTTP bajo `/api`.
- JWT: JSON Web Token usado para autenticacion.
- ROM: archivo de juego (por ejemplo `.nds`).
- SRS: Software Requirements Specification.

### 1.4 Referencias
- Codigo fuente del proyecto (este repositorio).
- `README.md` para variables de entorno y ejecucion local.

### 1.5 Vision general del documento
El documento describe el producto, sus interfaces externas, requisitos funcionales, restricciones y requisitos no funcionales minimos.

## 2. Descripcion general

### 2.1 Perspectiva del producto
Aplicacion web compuesta por:
- Frontend: archivos estaticos servidos por el backend (ruta `/`).
- Backend: servicio HTTP en Go (Fiber) exponiendo API REST.
- Persistencia: PostgreSQL (requerida via `DATABASE_URL`).

### 2.2 Funciones del producto (alto nivel)
- Registro y login de usuarios.
- Emision y verificacion de token JWT para endpoints protegidos.
- Descarga de ROMs (publica) y manejo de ROMs (protegido).
- CRUD de saves (protegido).
- Operaciones de Nuzlocke sobre una save (protegido).

### 2.3 Caracteristicas de usuario
- Usuario anonimo: puede registrarse e iniciar sesion; puede descargar ROMs (segun implementacion actual).
- Usuario autenticado: accede a endpoints protegidos y a la UI posterior al login.

### 2.4 Restricciones
- `DATABASE_URL` es obligatoria para ejecutar backend.
- `JWT_SECRET` es opcional; si no se define, se usa un valor por defecto (riesgo de seguridad en entornos reales).
- El backend sirve frontend desde `FRONTEND_DIR` (default) u override por variable de entorno.
- Directorio de ROMs configurable por `ROMS_DIR`.

### 2.5 Suposiciones y dependencias
- Se asume una instancia PostgreSQL accesible desde el backend.
- Se asume navegador moderno para ejecutar JS del frontend.
- Para desarrollo local sin instalar Postgres, se usa Docker Compose (segun `README.md`).

## 3. Requisitos especificos

### 3.1 Interfaces externas

#### 3.1.1 Interfaz de usuario
- Pagina de autenticacion: `/` (`frontend/index.html`).
- Lobby: `/lobby.html` (acceso esperado tras login).
- Pantalla de juego: `/play.html` (segun archivos de frontend).

#### 3.1.2 Interfaz de hardware
- No aplica (cliente es navegador; servidor es host con acceso a red y disco).

#### 3.1.3 Interfaz de software
- PostgreSQL via `DATABASE_URL`.
- HTTP API en el mismo origen (`window.location.origin + '/api'`).

#### 3.1.4 Interfaz de comunicaciones
- HTTP/HTTPS (segun despliegue). En local: `http://localhost:3030` (default).

### 3.2 Requisitos funcionales

Nota: los endpoints listados se derivan de `backend/main.go`.

#### RF-1 Registro de usuario
- El sistema debe permitir registrar un usuario via `POST /api/register` con `username` y `password`.
- El sistema debe rechazar usernames duplicados.

#### RF-2 Inicio de sesion
- El sistema debe autenticar via `POST /api/login` y devolver un token JWT y el username.
- El sistema debe rechazar credenciales invalidas.

#### RF-3 Autorizacion por token
- Los endpoints protegidos deben requerir `Authorization: Bearer <token>`.
- Si el token falta o es invalido, debe responder 401.

#### RF-4 Descarga publica de ROM
- El sistema debe exponer `GET /api/roms/download` sin requerir autenticacion.

#### RF-5 Gestion de ROMs (protegido)
- `GET /api/roms` lista ROMs para usuario autenticado.
- `POST /api/roms/upload` permite subir ROMs para usuario autenticado.

#### RF-6 Gestion de saves (protegido)
- `GET /api/saves` lista saves del usuario autenticado.
- `POST /api/saves` crea una save.
- `GET /api/saves/:id` obtiene datos de guardado para una save del usuario.
- `PUT /api/saves/:id` actualiza datos de guardado.
- `DELETE /api/saves/:id` elimina una save.

#### RF-7 Nuzlocke (protegido)
- `GET /api/saves/:id/nuzlocke` obtiene datos de nuzlocke de la save.
- `POST /api/saves/:id/nuzlocke` agrega un pokemon.
- `PUT /api/saves/:id/nuzlocke/:pokemonId/kill` marca un pokemon como muerto.
- `DELETE /api/saves/:id/nuzlocke/:pokemonId` elimina un pokemon.

### 3.3 Requisitos de performance
- El backend configura `BodyLimit` en 64MB (limite requerido para ROMs y saves grandes).

### 3.4 Restricciones de diseno
- Backend implementado en Go con Fiber.
- Autenticacion basada en JWT (HS256).

### 3.5 Atributos del sistema (no funcionales)
- Seguridad: `JWT_SECRET` debe ser configurable; el valor por defecto no es adecuado para produccion.
- Portabilidad: se provee `Dockerfile` para construir/ejecutar imagen.
- Operabilidad: variables `PORT`, `ROMS_DIR`, `FRONTEND_DIR`, `MIGRATIONS_DIR`.

### 3.6 Otros requisitos
- Migraciones: el backend aplica migraciones SQL al iniciar y opcionalmente via flags.

## 4. Apendices

### 4.1 Variables de entorno (observadas)
- `DATABASE_URL` (obligatoria)
- `JWT_SECRET` (opcional)
- `PORT` (opcional)
- `ROMS_DIR` (opcional)
- `FRONTEND_DIR` (opcional)
- `MIGRATIONS_DIR` (opcional)

