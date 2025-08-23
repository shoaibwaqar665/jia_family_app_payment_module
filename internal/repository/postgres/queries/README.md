# SQL Queries

This directory contains SQL query files that define the database operations for the payment service.

## Query Files

### plans.sql
Contains queries for managing subscription plans:
- `GetPlanByID` - Retrieve a plan by ID (active only)
- `ListActivePlans` - List all active plans
- `InsertPlan` - Create a new plan
- `UpdatePlanActive` - Update plan active status

### entitlements.sql
Contains queries for managing user entitlements:
- `CheckEntitlement` - Check if user has active entitlement for a feature
- `ListEntitlementsByUser` - List all entitlements for a user
- `InsertEntitlement` - Create a new entitlement
- `UpdateEntitlementStatus` - Update entitlement status
- `UpdateEntitlementExpiry` - Update entitlement expiration
- `GetEntitlementByID` - Get entitlement by ID
- `ListExpiringEntitlements` - List entitlements that are expiring

## Query Naming Conventions

- Use descriptive names that indicate the operation and entity
- Follow the pattern: `{Operation}{Entity}{By}{Field}` (e.g., `GetPlanByID`)
- Use `:one` for single row results, `:many` for multiple rows, `:exec` for no results

## SQL Features Used

- `sqlc.arg()` for required parameters
- `sqlc.narg()` for nullable parameters
- Proper use of `ORDER BY` and `LIMIT` clauses
- Efficient indexing considerations in WHERE clauses
