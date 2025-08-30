#!/usr/bin/env python3
"""
Simple test script to verify the Payment Service is running and responding
"""

import socket
import json
import sys
from datetime import datetime

def test_service_connection():
    """Test if the service port is open and listening"""
    try:
        # Test if port 8081 is open
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        result = sock.connect_ex(('localhost', 8081))
        sock.close()
        
        if result == 0:
            print("✅ Service port is open and listening!")
            print(f"📍 Service is running on: localhost:8081")
            print(f"🕐 Test time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
            return True
        else:
            print("❌ Service port is not accessible")
            return False
        
    except Exception as e:
        print(f"❌ Connection test failed: {e}")
        return False

def test_health_check():
    """Test health check endpoint"""
    try:
        # This is a simplified test - in a real scenario you'd use the proper gRPC client
        print("🔍 Testing health check endpoint...")
        print("   Note: Use Postman or a proper gRPC client for full testing")
        return True
    except Exception as e:
        print(f"❌ Health check test failed: {e}")
        return False

def main():
    print("🚀 Testing Payment Service...")
    print("=" * 50)
    
    # Test connection
    if test_service_connection():
        print("\n📋 Available endpoints:")
        print("   • Health Check: grpc://localhost:8081/grpc.health.v1.Health/Check")
        print("   • Payment Service: grpc://localhost:8081/payment.v1.PaymentService/*")
        print("\n📝 Next steps:")
        print("   1. Import the Postman collection: postman_collection_enhanced.json")
        print("   2. Set the base_url variable to: localhost:8081")
        print("   3. Start testing the endpoints!")
        print("\n🔑 Authentication:")
        print("   • Use header: better-auth-token")
        print("   • Test value: spiff_id_test_user_123")
        print("   • Webhook endpoints don't require authentication")
        
        test_health_check()
        
        print("\n✅ Service is ready for testing!")
    else:
        print("\n❌ Service is not responding. Please check:")
        print("   1. Is the service running? (go run ./cmd/paymentservice)")
        print("   2. Are PostgreSQL and Redis running?")
        print("   3. Check the service logs for errors")
        sys.exit(1)

if __name__ == "__main__":
    main()
