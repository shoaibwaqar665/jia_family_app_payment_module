# Pricing Update Summary

**Date:** October 21, 2025  
**Database:** jia_family_app (Neon PostgreSQL)  
**Migration Status:** ✅ Successfully Applied

## Overview
Successfully updated the database plans according to the new Jia pricing structure from the pricing document.

## New Pricing Structure

### Subscription Plans

| Plan | Monthly Price | Yearly Price | Storage | Max Users | Best For |
|------|--------------|--------------|---------|-----------|----------|
| **BASIC** | FREE | FREE | 0.5GB | 1 | Users just starting out |
| **PLUS** | $0.99 | $9.50 (17% off) | 10GB | 1 | Individuals who need more space |
| **PRO** | $1.99 | $19.10 (17% off) | 50GB | 1 | Serious genealogists and researchers |
| **PRO+** | $2.99 | $28.70 (17% off) | Unlimited | 1 | Power users with extensive libraries |
| **FAMILY** | $9.99 | $95.90 (17% off) | Unlimited | 6 | Families building their tree together |

### One-Time Add-Ons

| Add-On | Price | Description |
|--------|-------|-------------|
| **Premium Template Pack** | $4.99 | Unlock premium templates |
| **Verify Profile** | $4.99 | Get verified to build trust |
| **Verify Group** | $9.99 | Verify your group (badge visible to you and invited members) |

## Features by Plan

### BASIC (Free Forever)
- 0.5GB storage
- Standard templates
- Core tree builder
- Basic features

### PLUS
- 10GB storage
- Premium templates
- Enhanced privacy controls
- Core tree builder
- Basic features

### PRO
- 50GB storage
- All PLUS features
- AI research tools
- Legal document templates

### PRO+
- Unlimited storage
- All PRO features
- AI research tools
- Legal document templates

### FAMILY
- Unlimited shared storage
- All PRO+ features
- Collaborative editing
- Up to 6 family members

## Database Changes

### Plans Inserted
✅ 12 plans successfully inserted:
- 1 FREE plan (BASIC)
- 8 subscription plans (4 tiers × 2 billing cycles)
- 3 one-time add-ons

### Entitlements Updated
✅ 3 existing entitlements migrated to `basic_free` plan

### Foreign Key Constraints
✅ Properly handled with CASCADE updates

## Migration Files Created

1. **0008_update_plans_to_new_pricing.sql** - Official migration file (up)
2. **0008_update_plans_to_new_pricing.down.sql** - Rollback migration
3. **apply_new_pricing_safe.sql** - Standalone SQL script (used for this update)

## Verification

Run this query to verify all plans:
```sql
SELECT id, name, price_cents, billing_cycle, max_users 
FROM plans 
ORDER BY 
  CASE 
    WHEN id = 'basic_free' THEN 1
    WHEN id LIKE 'plus%' THEN 2
    WHEN id LIKE 'pro%' AND id NOT LIKE 'pro_plus%' THEN 3
    WHEN id LIKE 'pro_plus%' THEN 4
    WHEN id LIKE 'family%' THEN 5
    ELSE 6
  END,
  billing_cycle;
```

## Next Steps

1. ✅ Database plans updated
2. ⏭️ Update frontend pricing display
3. ⏭️ Update API documentation
4. ⏭️ Update marketing materials
5. ⏭️ Test subscription flows with new pricing

## Notes

- All prices are in USD
- Yearly plans offer 17% discount
- Storage limits: -1 indicates unlimited storage
- Family plan supports up to 6 users
- One-time add-ons available on any paid plan

## Rollback

If you need to rollback to the old pricing:
```bash
psql "your-db-connection-string" -f migrations/0008_update_plans_to_new_pricing.down.sql
```

---

**Status:** Migration completed successfully ✅

