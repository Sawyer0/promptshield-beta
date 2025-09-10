import { Shield, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { SignUp as ClerkSignUp } from "@clerk/clerk-react";

interface SignUpProps {
  onBack: () => void;
  onSignIn?: () => void;
}

export default function SignUp({ onBack }: SignUpProps) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="flex justify-center mb-6">
          <Shield className="h-12 w-12 text-primary" />
        </div>
        {/* Clerk-hosted SignUp supports OAuth + email/password out of the box */}
        <ClerkSignUp
          signInUrl="/sign-in"
          afterSignUpUrl="/choose-tenant"
          redirectUrl="/choose-tenant"
          appearance={{ elements: { formButtonPrimary: 'bg-primary text-primary-foreground hover:opacity-90' } }}
        />
        <div className="mt-4 text-center">
          <Button
            type="button"
            variant="ghost"
            className="w-full"
            onClick={onBack}
            data-testid="button-back-to-signin"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Sign In
          </Button>
        </div>
      </div>
    </div>
  );
}
