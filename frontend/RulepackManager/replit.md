# Overview

This is a full-stack web application for PromptShield, an AI Security Management Platform that provides security management and threat protection for AI systems. The application features a React frontend with a Node.js/Express backend, built to manage security rulepacks, tenants, policy assignments, and audit logs for AI system protection.

# User Preferences

Preferred communication style: Simple, everyday language.

# System Architecture

## Frontend Architecture
- **Framework**: React 18 with TypeScript using Vite as the build tool
- **UI Library**: Radix UI components with shadcn/ui design system
- **Styling**: Tailwind CSS with CSS variables for theming
- **State Management**: TanStack Query (React Query) for server state management
- **Routing**: Wouter for client-side routing
- **Forms**: React Hook Form with Zod validation
- **Authentication**: Replit OAuth integration with session-based authentication

## Backend Architecture
- **Runtime**: Node.js with Express.js framework
- **Language**: TypeScript with ES modules
- **Database ORM**: Drizzle ORM for type-safe database operations
- **Session Management**: Express sessions with PostgreSQL storage
- **Authentication**: OpenID Connect (OIDC) with Passport.js strategy
- **API Design**: RESTful APIs with standardized error handling and logging

## Database Design
- **Primary Database**: PostgreSQL via Neon serverless connection
- **Schema Management**: Drizzle migrations with schema definitions in TypeScript
- **Key Tables**: 
  - Users (for authentication)
  - RulePacks (security rules and policies)
  - Tenants (organizational units)
  - Policy Assignments (linking rules to tenants)
  - Audit Events (activity logging)
  - Sessions (authentication state)

## Development Architecture
- **Monorepo Structure**: Client, server, and shared code in a single repository
- **Shared Types**: Common TypeScript schemas and types shared between frontend and backend
- **Hot Reloading**: Vite development server with Express middleware integration
- **Path Aliases**: TypeScript path mapping for clean imports (@/, @shared/)

## Security Features
- **Session-based Authentication**: Secure cookie-based sessions with PostgreSQL storage
- **CSRF Protection**: Built into session management
- **Input Validation**: Zod schemas for runtime type checking and validation
- **Environment Isolation**: Separate development and production configurations

## Data Flow
- Frontend makes authenticated requests to Express API endpoints
- Backend validates requests, interacts with PostgreSQL via Drizzle ORM
- Real-time updates achieved through React Query's cache invalidation
- Audit events automatically logged for all CRUD operations

# External Dependencies

## Database Services
- **Neon PostgreSQL**: Serverless PostgreSQL database hosting
- **Drizzle ORM**: Type-safe database client and migration tool

## Authentication Services
- **Replit OAuth**: Primary authentication provider using OpenID Connect
- **Passport.js**: Authentication middleware for Node.js

## UI and Styling
- **Radix UI**: Headless UI component primitives
- **shadcn/ui**: Pre-built component library based on Radix UI
- **Tailwind CSS**: Utility-first CSS framework
- **Lucide React**: Icon library

## Development Tools
- **Vite**: Frontend build tool and development server
- **TypeScript**: Type safety across the entire stack
- **ESBuild**: Backend bundling for production builds
- **TanStack Query**: Server state management and caching

## Production Deployment
- **Express Static Serving**: Frontend assets served by Express in production
- **Environment Variables**: Configuration through DATABASE_URL, SESSION_SECRET, etc.
- **Process Management**: Single Node.js process serving both frontend and API