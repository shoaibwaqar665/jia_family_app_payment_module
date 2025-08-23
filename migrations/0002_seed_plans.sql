-- Migration: 0002_seed_plans
-- Description: Seed plans table with active subscription plans

-- Insert basic individual plans
INSERT INTO plans (
    id, 
    name, 
    description, 
    feature_codes, 
    billing_cycle, 
    price_cents, 
    currency, 
    max_users, 
    usage_limits, 
    metadata, 
    active
) VALUES 
-- Basic Plan - Monthly
(
    'basic_monthly',
    'Basic Plan - Monthly',
    'Essential features for individual users with monthly billing',
    ARRAY['basic_storage', 'basic_support', 'core_features'],
    'monthly',
    999, -- $9.99
    'USD',
    1,
    '{"storage_gb": 10, "api_calls_per_month": 1000, "support_level": "email"}',
    '{"category": "individual", "popular": false, "recommended_for": "personal_use"}',
    true
),

-- Basic Plan - Yearly
(
    'basic_yearly',
    'Basic Plan - Yearly',
    'Essential features for individual users with yearly billing (save 20%)',
    ARRAY['basic_storage', 'basic_support', 'core_features'],
    'yearly',
    9590, -- $95.90 (20% discount)
    'USD',
    1,
    '{"storage_gb": 10, "api_calls_per_month": 1000, "support_level": "email"}',
    '{"category": "individual", "popular": false, "recommended_for": "personal_use", "savings_percent": 20}',
    true
),

-- Pro Plan - Monthly
(
    'pro_monthly',
    'Pro Plan - Monthly',
    'Professional features for power users with monthly billing',
    ARRAY['pro_storage', 'pro_support', 'core_features', 'advanced_analytics', 'api_access'],
    'monthly',
    1999, -- $19.99
    'USD',
    1,
    '{"storage_gb": 100, "api_calls_per_month": 10000, "support_level": "priority_email", "analytics_retention_days": 90}',
    '{"category": "individual", "popular": true, "recommended_for": "professionals"}',
    true
),

-- Pro Plan - Yearly
(
    'pro_yearly',
    'Pro Plan - Yearly',
    'Professional features for power users with yearly billing (save 20%)',
    ARRAY['pro_storage', 'pro_support', 'core_features', 'advanced_analytics', 'api_access'],
    'yearly',
    19190, -- $191.90 (20% discount)
    'USD',
    1,
    '{"storage_gb": 100, "api_calls_per_month": 10000, "support_level": "priority_email", "analytics_retention_days": 90}',
    '{"category": "individual", "popular": true, "recommended_for": "professionals", "savings_percent": 20}',
    true
),

-- Family Plan - Monthly
(
    'family_monthly',
    'Family Plan - Monthly',
    'Family-friendly plan with shared storage and features for up to 6 users',
    ARRAY['family_storage', 'family_support', 'core_features', 'family_sharing', 'parental_controls'],
    'monthly',
    2999, -- $29.99
    'USD',
    6,
    '{"storage_gb": 500, "api_calls_per_month": 25000, "support_level": "priority_email", "family_members": 6}',
    '{"category": "family", "popular": true, "recommended_for": "families", "max_children": 4}',
    true
),

-- Family Plan - Yearly
(
    'family_yearly',
    'Family Plan - Yearly',
    'Family-friendly plan with shared storage and features for up to 6 users (save 20%)',
    ARRAY['family_storage', 'family_support', 'core_features', 'family_sharing', 'parental_controls'],
    'yearly',
    28790, -- $287.90 (20% discount)
    'USD',
    6,
    '{"storage_gb": 500, "api_calls_per_month": 25000, "support_level": "priority_email", "family_members": 6}',
    '{"category": "family", "popular": true, "recommended_for": "families", "max_children": 4, "savings_percent": 20}',
    true
),

-- Enterprise Plan - Monthly
(
    'enterprise_monthly',
    'Enterprise Plan - Monthly',
    'Enterprise-grade features with unlimited storage and priority support',
    ARRAY['enterprise_storage', 'enterprise_support', 'core_features', 'advanced_analytics', 'api_access', 'sso', 'audit_logs', 'custom_integrations'],
    'monthly',
    9999, -- $99.99
    'USD',
    100,
    '{"storage_gb": -1, "api_calls_per_month": -1, "support_level": "dedicated", "analytics_retention_days": 365, "sso_providers": ["saml", "oidc"]}',
    '{"category": "enterprise", "popular": false, "recommended_for": "businesses", "min_users": 10}',
    true
),

-- Enterprise Plan - Yearly
(
    'enterprise_yearly',
    'Enterprise Plan - Yearly',
    'Enterprise-grade features with unlimited storage and priority support (save 20%)',
    ARRAY['enterprise_storage', 'enterprise_support', 'core_features', 'advanced_analytics', 'api_access', 'sso', 'audit_logs', 'custom_integrations'],
    'yearly',
    95990, -- $959.90 (20% discount)
    'USD',
    100,
    '{"storage_gb": -1, "api_calls_per_month": -1, "support_level": "dedicated", "analytics_retention_days": 365, "sso_providers": ["saml", "oidc"]}',
    '{"category": "enterprise", "popular": false, "recommended_for": "businesses", "min_users": 10, "savings_percent": 20}',
    true
),

-- EUR Pricing (Basic Pro Monthly)
(
    'basic_pro_monthly_eur',
    'Basic Pro Plan - Monthly (EUR)',
    'Professional features for European users with monthly billing',
    ARRAY['pro_storage', 'pro_support', 'core_features', 'advanced_analytics', 'api_access'],
    'monthly',
    1899, -- €18.99
    'EUR',
    1,
    '{"storage_gb": 100, "api_calls_per_month": 10000, "support_level": "priority_email", "analytics_retention_days": 90}',
    '{"category": "individual", "popular": true, "recommended_for": "professionals", "region": "europe"}',
    true
),

-- EUR Pricing (Family Monthly)
(
    'family_monthly_eur',
    'Family Plan - Monthly (EUR)',
    'Family-friendly plan for European users with shared storage and features',
    ARRAY['family_storage', 'family_support', 'core_features', 'family_sharing', 'parental_controls'],
    'monthly',
    2899, -- €28.99
    'EUR',
    6,
    '{"storage_gb": 500, "api_calls_per_month": 25000, "support_level": "priority_email", "family_members": 6}',
    '{"category": "family", "popular": true, "recommended_for": "families", "max_children": 4, "region": "europe"}',
    true
);

-- Create down migration
-- Migration: 0002_seed_plans_down
-- Description: Remove seeded plans data

-- Note: This is a destructive operation - use with caution in production
-- DELETE FROM plans WHERE id IN (
--     'basic_monthly', 'basic_yearly', 'pro_monthly', 'pro_yearly',
--     'family_monthly', 'family_yearly', 'enterprise_monthly', 'enterprise_yearly',
--     'basic_pro_monthly_eur', 'family_monthly_eur'
-- );
