-- ============================================
-- Apply New Jia Pricing Structure
-- ============================================
-- This script deletes existing plans and inserts the new pricing structure
-- Database: jia_family_app
-- Date: October 2025
-- ============================================

-- Delete all existing plans
DELETE FROM plans;

-- ============================================
-- SUBSCRIPTION PLANS
-- ============================================

-- BASIC - Free Forever
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'basic_free',
    'BASIC - Free Forever',
    'Free forever plan with basic storage and features for users just starting out',
    ARRAY['basic_storage', 'standard_templates', 'core_tree_builder', 'basic_features'],
    'monthly',
    0,
    'USD',
    1,
    '{"storage_gb": 0.5, "support_level": "community"}',
    '{"category": "individual", "popular": false, "recommended_for": "new_users", "free_forever": true}',
    true
);

-- PLUS - Monthly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'plus_monthly',
    'PLUS - Monthly',
    'Perfect for individuals who need more space with premium templates and enhanced privacy',
    ARRAY['plus_storage', 'premium_templates', 'enhanced_privacy', 'core_tree_builder', 'basic_features'],
    'monthly',
    99,
    'USD',
    1,
    '{"storage_gb": 10, "support_level": "email"}',
    '{"category": "individual", "popular": true, "recommended_for": "individual_users", "display_order": 1}',
    true
);

-- PLUS - Yearly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'plus_yearly',
    'PLUS - Yearly',
    'Perfect for individuals who need more space with premium templates and enhanced privacy (save 17%)',
    ARRAY['plus_storage', 'premium_templates', 'enhanced_privacy', 'core_tree_builder', 'basic_features'],
    'yearly',
    950,
    'USD',
    1,
    '{"storage_gb": 10, "support_level": "email"}',
    '{"category": "individual", "popular": true, "recommended_for": "individual_users", "display_order": 1, "savings_percent": 17}',
    true
);

-- PRO - Monthly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'pro_monthly',
    'PRO - Monthly',
    'For serious genealogists with AI research tools and legal document templates',
    ARRAY['pro_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features'],
    'monthly',
    199,
    'USD',
    1,
    '{"storage_gb": 50, "support_level": "priority_email"}',
    '{"category": "individual", "popular": true, "recommended_for": "serious_genealogists", "display_order": 2}',
    true
);

-- PRO - Yearly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'pro_yearly',
    'PRO - Yearly',
    'For serious genealogists with AI research tools and legal document templates (save 17%)',
    ARRAY['pro_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features'],
    'yearly',
    1910,
    'USD',
    1,
    '{"storage_gb": 50, "support_level": "priority_email"}',
    '{"category": "individual", "popular": true, "recommended_for": "serious_genealogists", "display_order": 2, "savings_percent": 17}',
    true
);

-- PRO+ - Monthly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'pro_plus_monthly',
    'PRO+ - Monthly',
    'For power users with unlimited storage and all advanced features',
    ARRAY['unlimited_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features'],
    'monthly',
    299,
    'USD',
    1,
    '{"storage_gb": -1, "support_level": "priority_email"}',
    '{"category": "individual", "popular": false, "recommended_for": "power_users", "display_order": 3}',
    true
);

-- PRO+ - Yearly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'pro_plus_yearly',
    'PRO+ - Yearly',
    'For power users with unlimited storage and all advanced features (save 17%)',
    ARRAY['unlimited_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features'],
    'yearly',
    2870,
    'USD',
    1,
    '{"storage_gb": -1, "support_level": "priority_email"}',
    '{"category": "individual", "popular": false, "recommended_for": "power_users", "display_order": 3, "savings_percent": 17}',
    true
);

-- FAMILY - Monthly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'family_monthly',
    'FAMILY - Monthly',
    'Perfect for families building their tree together with unlimited shared storage and collaborative editing',
    ARRAY['unlimited_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features', 'family_sharing', 'collaborative_editing'],
    'monthly',
    999,
    'USD',
    6,
    '{"storage_gb": -1, "support_level": "priority_email", "family_members": 6}',
    '{"category": "family", "popular": true, "recommended_for": "families", "display_order": 4}',
    true
);

-- FAMILY - Yearly
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'family_yearly',
    'FAMILY - Yearly',
    'Perfect for families building their tree together with unlimited shared storage and collaborative editing (save 17%)',
    ARRAY['unlimited_storage', 'premium_templates', 'enhanced_privacy', 'ai_research_tools', 'legal_templates', 'core_tree_builder', 'basic_features', 'family_sharing', 'collaborative_editing'],
    'yearly',
    9590,
    'USD',
    6,
    '{"storage_gb": -1, "support_level": "priority_email", "family_members": 6}',
    '{"category": "family", "popular": true, "recommended_for": "families", "display_order": 4, "savings_percent": 17}',
    true
);

-- ============================================
-- ONE-TIME ADD-ONS
-- ============================================

-- Premium Template Pack
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'premium_template_pack',
    'Premium Template Pack',
    'Unlock premium templates to make your family tree look stunning',
    ARRAY['premium_templates'],
    'one_time',
    499,
    'USD',
    1,
    '{"template_access": "premium"}',
    '{"category": "addon", "type": "one_time", "display_order": 5}',
    true
);

-- Verify Profile
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'verify_profile',
    'Verify Profile',
    'Get verified to build trust in your family network',
    ARRAY['profile_verification'],
    'one_time',
    499,
    'USD',
    1,
    '{"verification_type": "profile"}',
    '{"category": "addon", "type": "one_time", "display_order": 6}',
    true
);

-- Verify Group
INSERT INTO plans (
    id, name, description, feature_codes, billing_cycle, price_cents, 
    currency, max_users, usage_limits, metadata, active
) VALUES (
    'verify_group',
    'Verify Group',
    'Verify your group (verification badge visible only to you and invited members)',
    ARRAY['group_verification'],
    'one_time',
    999,
    'USD',
    1,
    '{"verification_type": "group", "visibility": "owner_and_invited"}',
    '{"category": "addon", "type": "one_time", "display_order": 7}',
    true
);

-- ============================================
-- Verification Query
-- ============================================
-- Run this to verify the plans were inserted correctly:
-- SELECT id, name, price_cents, billing_cycle, max_users FROM plans ORDER BY 
--   CASE 
--     WHEN id = 'basic_free' THEN 1
--     WHEN id LIKE 'plus%' THEN 2
--     WHEN id LIKE 'pro%' AND id NOT LIKE 'pro_plus%' THEN 3
--     WHEN id LIKE 'pro_plus%' THEN 4
--     WHEN id LIKE 'family%' THEN 5
--     ELSE 6
--   END,
--   billing_cycle;

