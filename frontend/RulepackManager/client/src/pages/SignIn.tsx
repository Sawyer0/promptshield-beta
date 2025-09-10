import { Shield, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { SignIn as ClerkSignIn } from '@clerk/clerk-react';

interface SignInProps {
  onBack: () => void;
  onSignUp: () => void;
}

export default function SignIn({ onBack, onSignUp }: SignInProps) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="flex justify-center mb-6">
          <Shield className="h-12 w-12 text-primary" />
        </div>
        {/* Use Clerk's hosted SignIn to support OAuth + password flows */}
        <ClerkSignIn
          signUpUrl="/sign-up"
          afterSignInUrl="/"
          redirectUrl="/"
          appearance={{ elements: { formButtonPrimary: 'bg-primary text-primary-foreground hover:opacity-90' } }}
        />
        <div className="mt-4 text-center">
          <Button 
            type="button"
            variant="ghost"
            className="w-full"
            onClick={onBack}
            data-testid="button-back-to-landing"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Landing
          </Button>
        </div>
      </div>
    </div>
  );
}
