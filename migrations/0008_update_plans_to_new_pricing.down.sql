-- Migration: 0008_update_plans_to_new_pricing_down
-- Description: Revert to old pricing structure

-- Delete new plans
DELETE FROM plans WHERE id IN (
    'basic_free',
    'plus_monthly',
    'plus_yearly',
    'pro_monthly',
    'pro_yearly',
    'pro_plus_monthly',
    'pro_plus_yearly',
    'family_monthly',
    'family_yearly',
    'premium_template_pack',
    'verify_profile',
    'verify_group'
);

-- Re-insert old plans
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
);

