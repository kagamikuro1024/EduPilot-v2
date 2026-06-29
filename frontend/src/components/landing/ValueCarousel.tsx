"use client";

import React, { useCallback } from "react";
import useEmblaCarousel from "embla-carousel-react";
import Autoplay from "embla-carousel-autoplay";
import { ChevronLeft, ChevronRight, Clock, Sparkles } from "lucide-react";

export default function ValueCarousel() {
  const [emblaRef, emblaApi] = useEmblaCarousel({ loop: true }, [
    Autoplay({ delay: 5000, stopOnInteraction: false }),
  ]);

  const scrollPrev = useCallback(() => {
    if (emblaApi) emblaApi.scrollPrev();
  }, [emblaApi]);

  const scrollNext = useCallback(() => {
    if (emblaApi) emblaApi.scrollNext();
  }, [emblaApi]);

  const slides = [
    {
      type: "old",
      title: "The Old Way",
      icon: <Clock className="w-10 h-10 text-gray-400" />,
      highlight: "2–3 days",
      description: "Wait 2-3 days for an answer to a simple question. Email ping-pong destroys momentum.",
      subtext: "Academic friction creates learning burnout.",
      bgColor: "bg-gray-50",
      borderColor: "border-gray-200",
      textColor: "text-gray-500",
    },
    {
      type: "new",
      title: "The EduPilot Way",
      icon: <Sparkles className="w-10 h-10 text-vibrant-primary" />,
      highlight: "< 1.5 seconds",
      description: "Instant responses in under 1.5 seconds directly from your authorized materials.",
      subtext: "Stay in the flow and master topics seamlessly.",
      bgColor: "bg-indigo-50/50",
      borderColor: "border-indigo-100",
      textColor: "text-vibrant-primary",
    },
  ];

  return (
    <section className="py-24 bg-soft-canvas relative">
      <div className="max-w-7xl mx-auto px-6">
        <div className="text-center mb-12">
          <h2 className="text-sm font-bold uppercase tracking-widest text-vibrant-primary/80 font-cohere-mono mb-3">
            Efficiency Boost
          </h2>
          <h3 className="text-4xl md:text-5xl font-cohere-display font-bold text-heading-dark">
            Transform your learning speed
          </h3>
        </div>

        {/* Carousel Container */}
        <div className="relative group max-w-4xl mx-auto">
          <div className="overflow-hidden rounded-3xl border border-accent-glow/50 bg-white shadow-xl shadow-indigo-500/5" ref={emblaRef}>
            <div className="flex">
              {slides.map((slide, index) => (
                <div className="flex-[0_0_100%] min-w-0" key={index}>
                  <div className={`p-8 md:p-16 flex flex-col items-center text-center ${slide.bgColor}`}>
                    <div className="mb-6 p-4 rounded-2xl bg-white shadow-sm border border-inherit">
                      {slide.icon}
                    </div>
                    <h4 className={`font-cohere-mono text-sm font-bold uppercase mb-2 ${slide.textColor}`}>
                      {slide.title}
                    </h4>
                    <div className="text-5xl md:text-7xl font-cohere-display font-bold text-heading-dark tracking-tight mb-6">
                      {slide.highlight}
                    </div>
                    <p className="text-xl md:text-2xl font-medium text-heading-dark/80 max-w-2xl mb-4 leading-relaxed">
                      {slide.description}
                    </p>
                    <p className="text-lg text-heading-dark/60 italic font-cohere-body">
                      {slide.subtext}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Navigation Controls */}
          <button
            className="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white shadow-lg border border-gray-100 text-heading-dark/50 hover:text-vibrant-primary hover:scale-110 transition-all md:-left-6 z-10 opacity-0 group-hover:opacity-100 duration-300"
            onClick={scrollPrev}
            aria-label="Previous slide"
          >
            <ChevronLeft className="w-6 h-6" />
          </button>
          <button
            className="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-white shadow-lg border border-gray-100 text-heading-dark/50 hover:text-vibrant-primary hover:scale-110 transition-all md:-right-6 z-10 opacity-0 group-hover:opacity-100 duration-300"
            onClick={scrollNext}
            aria-label="Next slide"
          >
            <ChevronRight className="w-6 h-6" />
          </button>
        </div>
      </div>
    </section>
  );
}
