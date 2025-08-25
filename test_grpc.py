#!/usr/bin/env python3
"""
Simple gRPC client for testing the Payment Service
Requires: pip install grpcio grpcio-tools
"""

import grpc
import json
import base64
from datetime import datetime

# Note: You'll need to generate the gRPC client code first
# protoc --python_out=. --grpc_python_out=. proto/payment/v1/payment_service.proto

def test_health_check():
    """Test the health check endpoint"""
    try:
        # Create insecure channel
        channel = grpc.insecure_channel('localhost:8081')
        
        # Import the generated health check service
        # from grpc_health.v1 import health_pb2, health_pb2_grpc
        
        # stub = health_pb2_grpc.HealthStub(channel)
        # response = stub.Check(health_pb2.HealthCheckRequest())
        # print(f"Health Check Response: {response.status}")
        
        print("Health check endpoint available at: grpc://localhost:8081/grpc.health.v1.Health/Check")
        print("Use a gRPC client like grpcurl or BloomRPC to test")
        
    except Exception as e:
        print(f"Error testing health check: {e}")

def test_payment_service():
    """Test the payment service endpoints"""
    print("\n=== Payment Service Endpoints ===")
    print("Base URL: grpc://localhost:8081")
    print("\nAvailable endpoints:")
    
    endpoints = [
        {
            "name": "Check Entitlement",
            "method": "payment.v1.PaymentService/CheckEntitlement",
            "description": "Check if user has access to a feature",
            "sample_request": {
                "user_id": "user123",
                "feature_code": "premium_feature"
            }
        },
        {
            "name": "List User Entitlements",
            "method": "payment.v1.PaymentService/ListUserEntitlements",
            "description": "List all entitlements for a user",
            "sample_request": {
                "user_id": "user123"
            }
        },
        {
            "name": "Create Checkout Session",
            "method": "payment.v1.PaymentService/CreateCheckoutSession",
            "description": "Create checkout session for a plan",
            "sample_request": {
                "plan_id": "basic_monthly",
                "user_id": "user123"
            }
        },
        {
            "name": "Payment Success Webhook",
            "method": "payment.v1.PaymentService/PaymentSuccessWebhook",
            "description": "Handle payment success (no auth required)",
            "sample_request": {
                "payload": "base64_encoded_webhook_data",
                "signature": "webhook_signature"
            }
        }
    ]
    
    for endpoint in endpoints:
        print(f"\n{endpoint['name']}")
        print(f"  Method: {endpoint['method']}")
        print(f"  Description: {endpoint['description']}")
        print(f"  Sample Request: {json.dumps(endpoint['sample_request'], indent=2)}")

def test_with_grpcurl():
    """Show how to test with grpcurl"""
    print("\n=== Testing with grpcurl ===")
    print("Install grpcurl: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest")
    print("\nCommands to test:")
    
    commands = [
        "# List available services",
        "grpcurl -plaintext localhost:8081 list",
        "",
        "# List methods in PaymentService",
        "grpcurl -plaintext localhost:8081 list payment.v1.PaymentService",
        "",
        "# Test health check",
        "grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check",
        "",
        "# Test CheckEntitlement (with auth)",
        'grpcurl -plaintext -H "better-auth-token: spiff_id_test_user_123" -d \'{"user_id": "user123", "feature_code": "premium_feature"}\' localhost:8081 payment.v1.PaymentService/CheckEntitlement',
        "",
        "# Test ListUserEntitlements (with auth)",
        'grpcurl -plaintext -H "better-auth-token: spiff_id_test_user_123" -d \'{"user_id": "user123"}\' localhost:8081 payment.v1.PaymentService/ListUserEntitlements',
        "",
        "# Test CreateCheckoutSession (with auth)",
        'grpcurl -plaintext -H "better-auth-token: spiff_id_test_user_123" -d \'{"plan_id": "basic_monthly", "user_id": "user123"}\' localhost:8081 payment.v1.PaymentService/CreateCheckoutSession'
    ]
    
    for cmd in commands:
        print(cmd)

def main():
    """Main test function"""
    print("=== Payment Service gRPC API Testing ===")
    print(f"Test started at: {datetime.now()}")
    
    test_health_check()
    test_payment_service()
    test_with_grpcurl()
    
    print("\n=== Testing Complete ===")
    print("\nTo test the API:")
    print("1. Make sure the service is running: go run ./cmd/paymentservice")
    print("2. Use grpcurl commands above or import the Postman collection")
    print("3. The service runs on localhost:8081")

if __name__ == "__main__":
    main()
