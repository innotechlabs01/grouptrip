# grouptrip

Monorepo backend Go + PWA Next.js

## Ramas
- `develop` → desarrollo
- `qa` → despliegue QA en Vercel
- `prd` → producción

Flujo: develop → qa → prd con PR obligatorio.

## CI/CD
- `.github/workflows/ci.yml` ejecuta lint + tests con cobertura ≥90% para backend y PWA.
- `.github/workflows/cd-qa.yml` despliega a Vercel QA cuando `qa` está verde.
- Despliegue a producción solo desde `prd` con PR aprobado y CI verde.

## Calidad
Ver `CONSTRAINTS.md` y `docs/CONTRIBUTING.md`.
