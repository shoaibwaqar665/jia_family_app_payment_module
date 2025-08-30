# Payment Service System Diagrams

## 🏗️ High-Level Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        A[Client App]
        B[Admin Panel]
        C[Webhook Source]
    end
    
    subgraph "Payment Service"
        D[gRPC Transport Layer]
        E[Use Case Layer]
        F[Repository Layer]
    end
    
    subgraph "Infrastructure"
        G[PostgreSQL Database]
        H[Redis Cache]
        I[Kafka Event Bus]
    end
    
    A -->|gRPC| D
    B -->|gRPC| D
    C -->|HTTP Webhook| D
    
    D --> E
    E --> F
    
    F --> G
    E --> H
    E --> I
```

## 💳 Payment Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant T as Transport Layer
    participant U as Use Case
    participant R as Repository
    participant D as Database
    participant S as Stripe
    
    C->>T: CreatePayment Request
    T->>U: Validate & Process
    U->>R: Save Payment (pending)
    R->>D: INSERT payment
    D-->>R: Payment ID
    R-->>U: Payment Created
    U->>S: Process Payment
    S-->>U: Payment Result
    U->>R: Update Status
    R->>D: UPDATE payment
    U->>T: Return Response
    T-->>C: Payment Created
```

## 🎫 Entitlement System

```mermaid
graph LR
    subgraph "Entitlement Flow"
        A[Payment Success] --> B[Webhook Handler]
        B --> C[Create Entitlement]
        C --> D[Save to Database]
        D --> E[Update Cache]
        E --> F[Publish Event]
    end
    
    subgraph "Access Check"
        G[Feature Request] --> H[Check Cache]
        H -->|Hit| I[Return Access]
        H -->|Miss| J[Query Database]
        J --> K[Update Cache]
        K --> I
    end
```

## 💰 Pricing Zone System

```mermaid
graph TD
    A[User Location] --> B[Get Country Code]
    B --> C[Find Pricing Zone]
    C --> D{Zone Type}
    
    D -->|Zone A| E[Premium - 1.00x]
    D -->|Zone B| F[Mid-High - 0.70x]
    D -->|Zone C| G[Mid-Low - 0.40x]
    D -->|Zone D| H[Low-Income - 0.20x]
    
    E --> I[Apply Multiplier]
    F --> I
    G --> I
    H --> I
    
    I --> J[Return Adjusted Price]
```

## 🗄️ Database Schema

```mermaid
erDiagram
    PLANS {
        string id PK
        string name
        text description
        text[] feature_codes
        string billing_cycle
        int price_cents
        string currency
        int max_users
        jsonb usage_limits
        jsonb metadata
        boolean active
        timestamp created_at
        timestamp updated_at
    }
    
    ENTITLEMENTS {
        uuid id PK
        string user_id
        string family_id
        string feature_code
        string plan_id FK
        string subscription_id
        string status
        timestamp granted_at
        timestamp expires_at
        jsonb usage_limits
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }
    
    PAYMENTS {
        uuid id PK
        int amount
        string currency
        string status
        string payment_method
        string customer_id
        string order_id
        text description
        string external_payment_id
        text failure_reason
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }
    
    PRICING_ZONES {
        uuid id PK
        string country
        string iso_code
        string zone
        string zone_name
        string world_bank_classification
        string gni_per_capita_threshold
        decimal pricing_multiplier
        timestamp created_at
        timestamp updated_at
    }
    
    PLANS ||--o{ ENTITLEMENTS : "has"
```

## 🔐 Authentication Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant I as gRPC Interceptor
    participant V as Auth Validator
    participant S as Service
    
    C->>I: Request + Auth Token
    I->>V: Validate Token
    V-->>I: User ID
    I->>S: Request + User Context
    S-->>I: Response
    I-->>C: Response
```

## 📡 Event System

```mermaid
graph TB
    subgraph "Event Sources"
        A[Payment Created]
        B[Payment Completed]
        C[Entitlement Granted]
        D[Entitlement Expired]
    end
    
    subgraph "Event Publisher"
        E[Kafka Publisher]
    end
    
    subgraph "Event Consumers"
        F[Notification Service]
        G[Analytics Service]
        H[Audit Service]
    end
    
    A --> E
    B --> E
    C --> E
    D --> E
    
    E --> F
    E --> G
    E --> H
```

## 🏛️ Clean Architecture Layers

```mermaid
graph TB
    subgraph "External Interfaces"
        A[gRPC API]
        B[Webhooks]
        C[Health Checks]
    end
    
    subgraph "Transport Layer"
        D[Payment Transport]
        E[Entitlement Transport]
        F[Checkout Transport]
    end
    
    subgraph "Use Case Layer"
        G[Payment Use Case]
        H[Entitlement Use Case]
        I[Checkout Use Case]
    end
    
    subgraph "Domain Layer"
        J[Payment Models]
        K[Entitlement Models]
        L[Pricing Zone Models]
    end
    
    subgraph "Repository Layer"
        M[Payment Repository]
        N[Entitlement Repository]
        O[Pricing Zone Repository]
    end
    
    subgraph "Infrastructure"
        P[PostgreSQL]
        Q[Redis]
        R[Kafka]
    end
    
    A --> D
    B --> D
    C --> D
    
    D --> G
    E --> H
    F --> I
    
    G --> J
    H --> K
    I --> L
    
    J --> M
    K --> N
    L --> O
    
    M --> P
    N --> P
    O --> P
    
    G --> Q
    H --> Q
    
    G --> R
    H --> R
```

## 🔄 Request Flow

```mermaid
flowchart TD
    A[Client Request] --> B{Authentication}
    B -->|Valid| C[Extract User Context]
    B -->|Invalid| D[Return Unauthorized]
    
    C --> E[Validate Request]
    E -->|Valid| F[Process Business Logic]
    E -->|Invalid| G[Return Bad Request]
    
    F --> H[Check Cache]
    H -->|Hit| I[Return Cached Data]
    H -->|Miss| J[Query Database]
    
    J --> K[Update Cache]
    K --> L[Publish Event]
    L --> M[Return Response]
    
    I --> M
    M --> N[Log Request]
```

## 📊 Service Health Monitoring

```mermaid
graph TB
    subgraph "Health Checks"
        A[Database Health]
        B[Redis Health]
        C[Kafka Health]
    end
    
    subgraph "Health Server"
        D[gRPC Health Server]
    end
    
    subgraph "Monitoring"
        E[Health Status]
        F[Service Discovery]
        G[Load Balancer]
    end
    
    A --> D
    B --> D
    C --> D
    
    D --> E
    E --> F
    F --> G
```

## 🚀 Deployment Architecture

```mermaid
graph TB
    subgraph "Load Balancer"
        A[Nginx/HAProxy]
    end
    
    subgraph "Payment Service Instances"
        B[Instance 1]
        C[Instance 2]
        D[Instance N]
    end
    
    subgraph "Shared Infrastructure"
        E[PostgreSQL Cluster]
        F[Redis Cluster]
        G[Kafka Cluster]
    end
    
    A --> B
    A --> C
    A --> D
    
    B --> E
    B --> F
    B --> G
    
    C --> E
    C --> F
    C --> G
    
    D --> E
    D --> F
    D --> G
```

---

## 📝 How to Render These Diagrams

### Option 1: GitHub
- GitHub automatically renders Mermaid diagrams in markdown files
- Just view this file on GitHub

### Option 2: Mermaid Live Editor
- Go to https://mermaid.live/
- Copy and paste any diagram code
- Export as PNG/SVG

### Option 3: VS Code Extension
- Install "Mermaid Preview" extension
- Open this file and use the preview

### Option 4: Command Line
```bash
# Install mermaid-cli
npm install -g @mermaid-js/mermaid-cli

# Generate PNG
mmdc -i SYSTEM_DIAGRAMS.md -o diagrams.png

# Generate SVG
mmdc -i SYSTEM_DIAGRAMS.md -o diagrams.svg
```

---

*These diagrams provide a comprehensive visual representation of the Payment Service architecture, data flow, and system interactions.*
