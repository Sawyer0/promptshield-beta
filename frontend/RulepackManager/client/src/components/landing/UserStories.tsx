import { Shield, FileCheck, Code, Building, ArrowRight, CheckCircle2, Sparkles, Users, Zap, Target } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useState } from "react";

export function UserStories() {
  const [activePersona, setActivePersona] = useState(0);

  const personas = [
    {
      icon: Shield,
      title: "Security Engineers",
      subtitle: "Implement AI security without complexity",
      painPoint: "We need AI security but don't know how to implement it",
      solution: "Zero-code integration via egress proxy with real-time scanning and visual policy management",
      benefits: ["No developer changes needed", "3-tier detection engine", "Automated compliance evidence"],
      stats: "99.9% Uptime"
    },
    {
      icon: FileCheck,
      title: "Compliance Officers", 
      subtitle: "Generate audit-ready evidence automatically",
      painPoint: "How do we prove our AI applications are secure?",
      solution: "Automated evidence collection with pre-built compliance mappings for SOC2, HIPAA, GDPR",
      benefits: ["Exportable audit reports", "Tamper-evident audit trails", "Pre-built framework mappings"],
      stats: "SOC2 Ready"
    },
    {
      icon: Code,
      title: "Developers",
      subtitle: "Ship AI features quickly and safely",
      painPoint: "Security requirements are slowing down our AI development",
      solution: "Low-code API integration with real-time feedback that doesn't block development",
      benefits: ["Simple HTTP calls", "Non-blocking security", "Clear feedback on issues"],
      stats: "< 5ms Latency"
    },
    {
      icon: Building,
      title: "CTOs & Engineering Leaders",
      subtitle: "Enable AI innovation without risk",
      painPoint: "We need to use AI but don't want to introduce security risks",
      solution: "Enterprise-grade architecture with scalable deployment and clear ROI",
      benefits: ["Multi-tenant support", "Future-proof security", "Clear business justification"],
      stats: "Enterprise Scale"
    }
  ];

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.15,
        delayChildren: 0.1
      }
    }
  };

  const itemVariants = {
    hidden: { 
      opacity: 0, 
      y: 30
    },
    visible: { 
      opacity: 1, 
      y: 0,
      transition: {
        duration: 0.6,
        ease: "easeOut"
      }
    }
  };

  return (
    <section className="marketing-container marketing-section">
      <motion.div
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-50px" }}
        variants={containerVariants}
      >
        {/* Header */}
        <motion.div className="text-center mb-12" variants={itemVariants}>
          <div
            className="inline-flex items-center gap-2 px-4 py-2 rounded-full border marketing-body font-medium mb-6"
            style={{ color: 'var(--brand-accent)', borderColor: 'var(--brand-accent)' }}
          >
            <Sparkles className="h-4 w-4" style={{ color: 'var(--brand-accent)' }} />
            Enterprise Solutions
          </div>
          <h2 className="marketing-h2 font-medium serif-display mb-6 text-foreground">
            Built for Every Role in Your Organization
          </h2>
          <p className="marketing-body text-foreground/80 max-w-3xl mx-auto">
            PoliSync Guard provides targeted solutions for every stakeholder involved in AI security and compliance
          </p>
        </motion.div>

        {/* Persona Navigation Tabs */}
        <motion.div className="mb-12" variants={itemVariants}>
          <div className="flex flex-wrap justify-center gap-3 p-3 bg-card border rounded-2xl">
            {personas.map((persona, index) => (
              <button
                key={persona.title}
                onClick={() => setActivePersona(index)}
                className={`px-6 py-3 rounded-xl marketing-body font-semibold border transition-all duration-300 ${
                  activePersona === index
                    ? 'text-background shadow-lg scale-105'
                    : 'text-muted-foreground hover:text-foreground bg-transparent'
                }`}
                style={activePersona === index ? { backgroundColor: 'var(--brand-accent)', borderColor: 'var(--brand-accent)' } : undefined}
              >
                <div className="flex items-center gap-2">
                  <persona.icon className="h-4 w-4" />
                  <span className="hidden sm:inline">{persona.title}</span>
                  <span className="sm:hidden">{persona.title.split(' ')[0]}</span>
                </div>
              </button>
            ))}
          </div>
        </motion.div>

        {/* Active Persona Display with smooth transitions */}
        <AnimatePresence mode="wait">
          {(() => {
            const persona = personas[activePersona];
            return (
              <motion.div
                key={persona.title}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.35 }}
                className="relative"
              >
                {/* Hero Card */}
                <div className="relative overflow-hidden rounded-2xl bg-marketing-card border border-border p-5 lg:p-6 mb-5">
                  <div className="grid lg:grid-cols-2 gap-5 lg:gap-6 items-center">
                    {/* Left Content */}
                  <div className="pl-2 md:pl-4 max-w-[560px]">
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-3 rounded-2xl bg-background shadow-sm">
                          <persona.icon className="h-12 w-12 text-foreground" />
                        </div>
                        <div>
                          <h3 className="marketing-h3 font-medium serif-display mb-2 text-foreground">{persona.title}</h3>
                          <p className="marketing-body text-foreground/80">{persona.subtitle}</p>
                        </div>
                      </div>

                      <button className="inline-flex items-center gap-2 px-6 py-3 rounded-lg text-background font-semibold transition-all duration-300 hover:scale-105 shadow-lg"
                              style={{ backgroundColor: 'var(--brand-accent)', borderColor: 'var(--brand-accent)' }}>
                        <span>Explore {persona.title} Solutions</span>
                        <ArrowRight className="h-5 w-5" />
                      </button>
                    </div>

                    {/* Right Content - Pain Point & Solution */}
                    <div className="space-y-4">
                      {/* Pain Point */}
                      <div className="p-4 rounded-xl bg-marketing-card border border-border">
                        <div className="flex items-center gap-3 mb-3">
                          <Target className="h-5 w-5 text-muted-foreground" />
                          <span className="text-sm font-bold uppercase tracking-wide text-muted-foreground">Pain Point</span>
                        </div>
                        <p className="marketing-body italic text-foreground">"{persona.painPoint}"</p>
                      </div>

                      {/* Solution */}
                      <div className="p-4 rounded-xl bg-marketing-card border border-border">
                        <div className="flex items-center gap-3 mb-3">
                          <Zap className="h-5 w-5 text-muted-foreground" />
                          <span className="text-sm font-bold uppercase tracking-wide text-muted-foreground">Our Solution</span>
                        </div>
                        <p className="marketing-body text-foreground">{persona.solution}</p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Benefits Grid */}
                <div className="grid md:grid-cols-3 gap-3">
                  {persona.benefits.map((benefit, i) => (
                    <motion.div
                      key={i}
                      className="p-4 rounded-xl bg-marketing-card border border-border hover:shadow-lg transition-all duration-300 hover:-translate-y-1"
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: i * 0.1 }}
                    >
                      <div className="flex items-start gap-3">
                        <CheckCircle2 className="h-6 w-6 text-muted-foreground mt-1 flex-shrink-0" />
                        <span className="marketing-body font-semibold text-foreground">{benefit}</span>
                      </div>
                    </motion.div>
                  ))}
                </div>
              </motion.div>
            );
          })()}
        </AnimatePresence>

        {/* Bottom CTA */}
        <motion.div 
          className="mt-20 pt-10 text-center border-t border-border"
          variants={itemVariants}
        >
          <div className="inline-flex items-center gap-2 px-8 py-4 rounded-2xl text-background font-semibold hover:opacity-95 transition-all duration-300 hover:scale-105 shadow-lg"
               style={{ backgroundColor: 'var(--brand-accent)', borderColor: 'var(--brand-accent)' }}>
            <Users className="h-5 w-5" />
            <span>Get Started for Your Organization</span>
            <ArrowRight className="h-5 w-5" />
          </div>
        </motion.div>
      </motion.div>
    </section>
  );
}
