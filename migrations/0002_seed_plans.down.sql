-- Migration: 0002_seed_plans_down
-- Description: Remove seeded plans data

-- Remove all seeded plans
DELETE FROM plans WHERE id IN (
    'basic_monthly', 'basic_yearly', 'pro_monthly', 'pro_yearly',
    'family_monthly', 'family_yearly', 'enterprise_monthly', 'enterprise_yearly',
    'basic_pro_monthly_eur', 'family_monthly_eur'
);
