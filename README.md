# eRateManager skeleton with refresh handler wired in

## Authentication & RBAC

This project uses Role-Based Access Control (RBAC) powered by [Casbin](https://casbin.org/).

### Roles

- **admin**: Full access to all resources.
- **editor**: Can read and write rates and providers.
- **viewer**: Can only read rates and providers.

### Authentication

The API supports Bearer Token authentication.

1.  **Login**: POST `/api/auth/login` with `{"username": "...", "password": "..."}`. Returns a session token.
2.  **API Tokens**: Manage tokens via `/api/auth/tokens`.

### Initial Setup

On first run, if no users exist, a default admin user is created:
- Username: `admin`
- Password: `admin`

**Change this password immediately!**

### API Usage

Include the token in the `Authorization` header:
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/rates/electric/cemc/residential
```
