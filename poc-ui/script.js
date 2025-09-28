// Configuration
const CONFIG = {
    API_BASE_URL: 'http://localhost:8082/api',
    STRIPE_PUBLISHABLE_KEY: 'pk_test_51PAonXSHVQMYbf3sSPa2xq688zX7w0c5PXJvKRhAE2uDXFz6wHNLQlskbY3pRIGPkEMEkbCYUCMbQfnIOVrMvV4v00P8vXcviw',
    AUTH_TOKEN: 'test_user_123',
    USER_ID: 'test_user_123'
};

// Plan configurations with database plan IDs
const PLANS = {
    basic_monthly: {
        name: 'Basic Plan',
        planId: 'basic_monthly', // Database plan ID
        basePrice: 9.99, // $9.99 in dollars
        currency: 'USD',
        features: ['basic_storage']
    },
    pro_monthly: {
        name: 'Pro Plan',
        planId: 'pro_monthly', // Database plan ID
        basePrice: 19.99, // $19.99 in dollars
        currency: 'USD',
        features: ['pro_storage']
    },
    family_monthly: {
        name: 'Family Plan',
        planId: 'family_monthly', // Database plan ID
        basePrice: 29.99, // $29.99 in dollars
        currency: 'USD',
        features: ['family_storage']
    }
};

// Pricing zones - will be loaded from database
let PRICING_ZONES = {};

// Global variables
let selectedPlan = null;
let stripe = null;
let elements = null;
let cardElement = null;

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    initializeStripe();
    setupEventListeners();
    loadPricingZones();
});

// Initialize Stripe
function initializeStripe() {
    stripe = Stripe(CONFIG.STRIPE_PUBLISHABLE_KEY);
    
    // Create elements instance
    elements = stripe.elements();
    
    // Create card element
    cardElement = elements.create('card', {
        style: {
            base: {
                fontSize: '16px',
                color: '#424770',
                '::placeholder': {
                    color: '#aab7c4',
                },
            },
        },
    });
}

// Load pricing zones from database
async function loadPricingZones() {
    try {
        const response = await fetch(`${CONFIG.API_BASE_URL}/pricing-zones`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Convert to the format expected by the UI
        PRICING_ZONES = {};
        data.pricing_zones.forEach(zone => {
            PRICING_ZONES[zone.iso_code] = {
                name: zone.zone_name,
                multiplier: zone.pricing_multiplier,
                country: zone.country,
                zone: zone.zone
            };
        });
        
        console.log('Loaded pricing zones:', PRICING_ZONES);
        
        // Update the country dropdown with all available countries
        updateCountryDropdown();
        
    } catch (error) {
        console.error('Failed to load pricing zones:', error);
        showToast('Failed to load pricing zones: ' + error.message, 'error');
        
        // Fallback to hardcoded zones if API fails
        PRICING_ZONES = {
            'US': { name: 'Premium', multiplier: 1.00, country: 'United States', zone: 'A' },
            'GB': { name: 'Premium', multiplier: 1.00, country: 'United Kingdom', zone: 'A' },
            'DE': { name: 'Premium', multiplier: 1.00, country: 'Germany', zone: 'A' },
            'CN': { name: 'Mid-High', multiplier: 0.70, country: 'China', zone: 'B' },
            'IN': { name: 'Mid-Low', multiplier: 0.40, country: 'India', zone: 'C' },
            'BR': { name: 'Mid-High', multiplier: 0.70, country: 'Brazil', zone: 'B' },
            'AF': { name: 'Low-Income', multiplier: 0.20, country: 'Afghanistan', zone: 'D' }
        };
        updateCountryDropdown();
    }
}

// Update country dropdown with available pricing zones
function updateCountryDropdown() {
    const countrySelect = document.getElementById('country');
    if (!countrySelect) return;
    
    // Clear existing options
    countrySelect.innerHTML = '';
    
    // Add options for each pricing zone
    Object.keys(PRICING_ZONES).forEach(isoCode => {
        const zone = PRICING_ZONES[isoCode];
        const option = document.createElement('option');
        option.value = isoCode;
        option.textContent = `${zone.country} (${zone.name} - ${(zone.multiplier * 100).toFixed(0)}%)`;
        countrySelect.appendChild(option);
    });
}

// Setup event listeners
function setupEventListeners() {
    // Plan selection buttons
    document.querySelectorAll('.select-plan-btn').forEach(button => {
        button.addEventListener('click', function() {
            const planId = this.getAttribute('data-plan');
            selectPlan(planId);
        });
    });
}

// Select a plan
function selectPlan(planId) {
    selectedPlan = planId;
    const plan = PLANS[planId];
    
    // Update UI
    document.getElementById('selectedPlanName').textContent = plan.name;
    document.getElementById('basePrice').textContent = formatPrice(plan.basePrice);
    
    // Update pricing based on current country
    updatePricing();
    
    // Show payment section
    document.getElementById('paymentSection').style.display = 'block';
    document.querySelector('.plans-section').style.display = 'none';
    
    // Scroll to payment section
    document.getElementById('paymentSection').scrollIntoView({ behavior: 'smooth' });
}

// Update pricing based on selected country
function updatePricing() {
    const country = document.getElementById('country').value;
    const zone = PRICING_ZONES[country];
    const plan = PLANS[selectedPlan];
    
    const adjustedPrice = plan.basePrice * zone.multiplier;
    
    // Update UI
    document.getElementById('pricingZone').textContent = `${zone.name} (${zone.multiplier}x)`;
    document.getElementById('totalPrice').textContent = formatPrice(adjustedPrice);
}

// Format price for display
function formatPrice(priceInDollars) {
    return `$${priceInDollars.toFixed(2)}`;
}

// Go back to plan selection
function goBack() {
    document.getElementById('paymentSection').style.display = 'none';
    document.querySelector('.plans-section').style.display = 'block';
    document.querySelector('.plans-section').scrollIntoView({ behavior: 'smooth' });
}

// Process payment
async function processPayment() {
    const payButton = document.getElementById('payButton');
    const payButtonText = document.getElementById('payButtonText');
    const payButtonLoading = document.getElementById('payButtonLoading');
    
    // Show loading state
    payButton.disabled = true;
    payButtonText.style.display = 'none';
    payButtonLoading.style.display = 'inline';
    
    try {
        // Get form data
        const country = document.getElementById('country').value;
        const familyId = document.getElementById('familyId').value;
        const plan = PLANS[selectedPlan];
        const zone = PRICING_ZONES[country];
        const adjustedPrice = plan.basePrice * zone.multiplier;
        
        // Create checkout session
        const sessionResponse = await createCheckoutSession({
            planId: plan.planId,
            userId: CONFIG.USER_ID,
            familyId: familyId || null,
            countryCode: country,
            basePrice: plan.basePrice,
            adjustedPrice: adjustedPrice,
            currency: plan.currency
        });
        
        if (sessionResponse.success) {
            // Redirect to Stripe Checkout URL directly
            window.location.href = sessionResponse.url;
        } else {
            throw new Error(sessionResponse.error || 'Failed to create checkout session');
        }
        
    } catch (error) {
        console.error('Payment error:', error);
        showToast('Payment failed: ' + error.message, 'error');
        
        // Reset button state
        payButton.disabled = false;
        payButtonText.style.display = 'inline';
        payButtonLoading.style.display = 'none';
    }
}

// Create checkout session
async function createCheckoutSession(data) {
    try {
        const response = await fetch(`${CONFIG.API_BASE_URL}/checkout`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'better-auth-token': CONFIG.AUTH_TOKEN
            },
            body: JSON.stringify({
                plan_id: data.planId,
                user_id: data.userId,
                family_id: data.familyId,
                country_code: data.countryCode,
                base_price: data.basePrice,
                currency: data.currency,
                success_url: `${window.location.origin}/success?session_id={CHECKOUT_SESSION_ID}&user_id=${data.userId}&plan_id=${data.planId}&family_id=${data.familyId}`,
                cancel_url: `${window.location.origin}/`
            })
        });

        if (!response.ok) {
            const errorData = await response.text();
            throw new Error(`HTTP ${response.status}: ${errorData}`);
        }

        const result = await response.json();
        
        return {
            success: true,
            sessionId: result.session_id,
            url: result.url
        };
        
    } catch (error) {
        console.error('Checkout session creation failed:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

// Simulate payment success (for POC)
function simulatePaymentSuccess(data, sessionId) {
    // Show success status
    showPaymentStatus('success', {
        sessionId: sessionId,
        planId: data.planId,
        amount: data.adjustedPrice,
        currency: data.currency,
        userId: data.userId
    });
    
    // Create entitlement
    createEntitlement(data, sessionId);
}

// Show payment status
function showPaymentStatus(status, details) {
    const statusSection = document.getElementById('statusSection');
    const statusIcon = document.getElementById('statusIcon');
    const statusTitle = document.getElementById('statusTitle');
    const statusMessage = document.getElementById('statusMessage');
    const statusDetails = document.getElementById('statusDetails');
    const statusButton = document.getElementById('statusButton');
    
    // Hide other sections
    document.getElementById('paymentSection').style.display = 'none';
    document.querySelector('.plans-section').style.display = 'none';
    
    // Show status section
    statusSection.style.display = 'block';
    
    if (status === 'success') {
        statusIcon.textContent = '✅';
        statusIcon.className = 'status-icon success';
        statusTitle.textContent = 'Payment Successful!';
        statusMessage.textContent = 'Your payment has been processed successfully. Your entitlement is being created...';
        
        statusDetails.innerHTML = `
            <h4>Payment Details</h4>
            <p><strong>Session ID:</strong> ${details.sessionId}</p>
            <p><strong>Plan:</strong> ${PLANS[details.planId].name}</p>
            <p><strong>Amount:</strong> ${formatPrice(details.amount)} ${details.currency}</p>
            <p><strong>User ID:</strong> ${details.userId}</p>
            <p><strong>Status:</strong> Completed</p>
        `;
        
        // Change button text and behavior for success
        statusButton.textContent = 'View Your Entitlements';
        statusButton.onclick = showEntitlements;
        statusButton.style.display = 'block';
        
    } else if (status === 'error') {
        statusIcon.textContent = '❌';
        statusIcon.className = 'status-icon error';
        statusTitle.textContent = 'Payment Failed';
        statusMessage.textContent = 'There was an error processing your payment.';
        
        statusDetails.innerHTML = `
            <h4>Error Details</h4>
            <p>${details.error || 'Unknown error occurred'}</p>
        `;
        
        // Keep original button behavior for errors
        statusButton.textContent = 'Start New Payment';
        statusButton.onclick = resetFlow;
        statusButton.style.display = 'block';
    }
    
    // Scroll to status section
    statusSection.scrollIntoView({ behavior: 'smooth' });
}

// Create entitlement (simulate webhook processing)
async function createEntitlement(data, sessionId) {
    try {
        // Simulate webhook processing
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        // First, call the webhook endpoint to process payment completion
        const webhookPayload = {
            session_id: sessionId,
            user_id: data.userId,
            plan_id: data.planId,
            feature_code: PLANS[data.planId].features[0], // Use first feature
            amount: data.adjustedPrice,
            currency: data.currency,
            status: 'completed',
            expires_at: null, // Lifetime for this POC
            metadata: {
                family_id: data.familyId,
                country_code: data.countryCode,
                base_price: data.basePrice,
                adjusted_price: data.adjustedPrice
            }
        };
        
        // Call webhook endpoint to process payment completion
        const webhookResponse = await fetch(`${CONFIG.API_BASE_URL}/webhook`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'better-auth-token': CONFIG.AUTH_TOKEN
            },
            body: JSON.stringify({
                payload: webhookPayload, // Send as object, not string
                signature: 'test_signature_' + sessionId, // Mock signature for POC
                provider: 'stripe' // Add provider field
            })
        });
        
        if (!webhookResponse.ok) {
            throw new Error(`Webhook failed: ${webhookResponse.status}`);
        }
        
        const webhookResult = await webhookResponse.json();
        console.log('Webhook processed:', webhookResult);
        
        showToast('Payment processed and entitlement created successfully! Redirecting to entitlements...', 'success');
        
        // Update status message to show completion
        const statusMessage = document.getElementById('statusMessage');
        if (statusMessage) {
            statusMessage.textContent = 'Your entitlement has been created successfully!';
        }
        
        // Redirect to entitlements page after successful creation
        setTimeout(() => {
            showEntitlements();
        }, 2000); // Wait 2 seconds to show the success message
        
    } catch (error) {
        console.error('Entitlement creation error:', error);
        showToast('Failed to process payment completion: ' + error.message, 'error');
    }
}

// Show entitlements
async function showEntitlements() {
    const entitlementsSection = document.getElementById('entitlementsSection');
    const entitlementsContainer = document.getElementById('entitlementsContainer');
    
    // Show loading state
    entitlementsContainer.innerHTML = `
        <div class="entitlement-card">
            <h3>Loading Entitlements...</h3>
            <p>Please wait while we fetch your entitlements...</p>
        </div>
    `;
    
    // Hide other sections
    document.querySelector('.plans-section').style.display = 'none';
    document.getElementById('paymentSection').style.display = 'none';
    document.getElementById('statusSection').style.display = 'none';
    
    // Show entitlements section
    entitlementsSection.style.display = 'block';
    entitlementsSection.scrollIntoView({ behavior: 'smooth' });
    
    // Show success message if coming from payment completion
    showToast('Welcome to your entitlements! Here are your active subscriptions.', 'success');
    
    try {
        // Call the API to get entitlements
        const response = await fetch(`${CONFIG.API_BASE_URL}/entitlements?user_id=${CONFIG.USER_ID}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'better-auth-token': CONFIG.AUTH_TOKEN
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        const entitlements = data.entitlements || [];
        
        if (entitlements.length === 0) {
            entitlementsContainer.innerHTML = `
                <div class="entitlement-card">
                    <h3>No Entitlements</h3>
                    <p>You don't have any active entitlements yet. Purchase a plan to get started!</p>
                </div>
            `;
        } else {
            entitlementsContainer.innerHTML = entitlements.map(entitlement => {
                // Get plan name from UUID
                const planName = getPlanNameFromUUID(entitlement.plan_id);
                
                // Convert timestamp to readable date
                const grantedAt = new Date(entitlement.granted_at.seconds * 1000);
                const expiresAt = entitlement.expires_at ? new Date(entitlement.expires_at.seconds * 1000) : null;
                
                return `
                    <div class="entitlement-card">
                        <h3>${planName}</h3>
                        <div class="entitlement-details">
                            <p><strong>Feature:</strong> ${entitlement.feature_code}</p>
                            <p><strong>Status:</strong> ${entitlement.status}</p>
                            <p><strong>Granted:</strong> ${grantedAt.toLocaleDateString()}</p>
                            <p><strong>Expires:</strong> ${expiresAt ? expiresAt.toLocaleDateString() : 'Never'}</p>
                            ${entitlement.subscription_id ? `<p><strong>Subscription ID:</strong> ${entitlement.subscription_id}</p>` : ''}
                            ${entitlement.family_id ? `<p><strong>Family ID:</strong> ${entitlement.family_id}</p>` : ''}
                        </div>
                        <span class="entitlement-status">${entitlement.status}</span>
                    </div>
                `;
            }).join('');
        }
        
    } catch (error) {
        console.error('Failed to fetch entitlements:', error);
        entitlementsContainer.innerHTML = `
            <div class="entitlement-card">
                <h3>Error Loading Entitlements</h3>
                <p>Failed to load entitlements: ${error.message}</p>
                <button onclick="showEntitlements()" class="btn-primary">Retry</button>
            </div>
        `;
    }
}

// Reset the flow
function resetFlow() {
    // Clear selections
    selectedPlan = null;
    
    // Reset form
    document.getElementById('familyId').value = '';
    document.getElementById('country').value = 'US';
    
    // Hide all sections except plans
    document.getElementById('paymentSection').style.display = 'none';
    document.getElementById('statusSection').style.display = 'none';
    document.getElementById('entitlementsSection').style.display = 'none';
    
    // Show plans section
    document.querySelector('.plans-section').style.display = 'block';
    document.querySelector('.plans-section').scrollIntoView({ behavior: 'smooth' });
}

// Show toast notification
function showToast(message, type = 'success') {
    const toastContainer = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    
    toastContainer.appendChild(toast);
    
    // Remove toast after 5 seconds
    setTimeout(() => {
        toast.remove();
    }, 5000);
}

// Generate random ID
function generateRandomId() {
    return Math.random().toString(36).substr(2, 9);
}

// Map UUID plan IDs back to plan names
function getPlanNameFromUUID(uuidPlanId) {
    // These are the deterministic UUIDs generated by the backend
    const uuidToPlanMap = {
        'f6845cd0-dda8-57af-8d1a-27a45d76b96a': 'Basic Plan', // basic_monthly
        'a1b2c3d4-e5f6-7890-abcd-ef1234567890': 'Pro Plan',   // pro_monthly (example)
        'b2c3d4e5-f6g7-8901-bcde-f23456789012': 'Family Plan' // family_monthly (example)
    };
    
    return uuidToPlanMap[uuidPlanId] || uuidPlanId;
}

// API helper functions
async function apiCall(endpoint, method = 'GET', data = null) {
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'better-auth-token': CONFIG.AUTH_TOKEN
        }
    };
    
    if (data) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(`${CONFIG.API_BASE_URL}${endpoint}`, options);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API call failed:', error);
        throw error;
    }
}

// Test API connectivity
async function testApiConnection() {
    try {
        // Test with a simple gRPC call (this would need to be adapted for HTTP)
        showToast('API connection test - this would test gRPC connectivity', 'warning');
    } catch (error) {
        showToast('API connection failed: ' + error.message, 'error');
    }
}

// Initialize API test on load
document.addEventListener('DOMContentLoaded', function() {
    // Load pricing zones
    loadPricingZones();
    
    // Check if we should show entitlements (from success page redirect)
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('show_entitlements') === 'true') {
        // Clear the URL parameter
        window.history.replaceState({}, document.title, window.location.pathname);
        // Show entitlements after a short delay to ensure page is loaded
        setTimeout(() => {
            showEntitlements();
        }, 500);
    }
    
    // Test API connection after a short delay
    setTimeout(testApiConnection, 1000);
});
