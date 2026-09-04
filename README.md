# grouptrip

Monorepo backend Go + PWA Next.js

## Ramas
- `develop` → desarrollo local, sin despliegue
- `qa` → despliegue QA en Vercel
- `prd` → producción

Flujo: develop → qa → prd con PR obligatorio.

## Vercel
- Production Branch: `prd`
- Preview Branches: `qa` únicamente
- Ignorar `develop` para evitar despliegues accidentales
- Variable: `NEXT_PUBLIC_API_URL` @grouptrip_api_url

## GitHub Secrets
- `VERCEL_TOKEN` → para workflow `cd-qa.yml`
- `AUTH_JWT_SECRET` → backend

## CI/CD
- `.github/workflows/ci.yml` ejecuta lint + tests con cobertura ≥90% para backend y PWA.
- `.github/workflows/cd-qa.yml` despliega a Vercel QA cuando `qa` está verde.
- Despliegue a producción solo desde `prd` con PR aprobado y CI verde.

## Calidad
Ver `CONSTRAINTS.md` y `docs/CONTRIBUTING.md`.
