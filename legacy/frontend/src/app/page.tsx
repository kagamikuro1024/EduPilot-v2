import Navbar from "@/components/landing/Navbar";
import HeroSection from "@/components/landing/HeroSection";
import ValueCarousel from "@/components/landing/ValueCarousel";
import SafetySection from "@/components/landing/SafetySection";
import TeamSection from "@/components/landing/TeamSection";
import Footer from "@/components/landing/Footer";

export default function RootPage() {
  return (
    <div className="edupilot-theme bg-soft-canvas min-h-screen text-heading-dark">
      <Navbar />
      <HeroSection />
      <ValueCarousel />
      <SafetySection />
      <TeamSection />
      <Footer />
    </div>
  );
}
