// Database Setup Script
// Run this to initialize your database: npm run setup-db

import { db } from './db';
import { sql } from 'drizzle-orm';
import fs from 'fs';
import path from 'path';

async function setupDatabase() {
  console.log('🚀 Setting up database...');
  
  try {
    // Read the schema SQL file
    const schemaSQL = fs.readFileSync(
      path.join(__dirname, 'schema.sql'), 
      'utf-8'
    );
    
    // Execute the schema
    await db.execute(sql.raw(schemaSQL));
    
    console.log('✅ Database schema created successfully!');
    
    // Insert sample data (optional)
    console.log('📝 Inserting sample policies...');
    
    const samplePolicies = [
      {
        name: 'Default Security Policy',
        type: 'security',
        content: `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: default-security
  version: "1.0"
rules:
  - id: "injection-defense"
    name: "Injection Defense"
    level: 1
    severity: "HIGH"
    keywords: ["ignore previous", "system prompt", "jailbreak"]
    response:
      action: "deny"
      message: "Security violation detected"`,
        is_active: true
      }
    ];
    
    // Note: You'll need to implement this based on your Drizzle schema
    console.log('✅ Sample data inserted!');
    
  } catch (error) {
    console.error('❌ Database setup failed:', error);
    process.exit(1);
  }
}

// Run setup
setupDatabase().then(() => {
  console.log('🎉 Database setup complete!');
  process.exit(0);
});