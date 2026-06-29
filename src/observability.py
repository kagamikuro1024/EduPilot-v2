from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer
import os
import logging

logger = logging.getLogger(__name__)

def setup_tracing():
    """Sets up OpenTelemetry tracing with OTLP exporter to Jaeger."""
    try:
        service_name = os.getenv("OTEL_SERVICE_NAME", "python-ai-orchestrator")
        endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4317")
        
        resource = Resource.create({"service.name": service_name})
        provider = TracerProvider(resource=resource)
        
        # Export to Jaeger using OTLP/gRPC
        # Wrapped in try-except to prevent crash if Jaeger is down (Scenario 2)
        otlp_exporter = OTLPSpanExporter(endpoint=endpoint, insecure=True)
        span_processor = BatchSpanProcessor(otlp_exporter)
        provider.add_span_processor(span_processor)
        
        trace.set_tracer_provider(provider)
        
        # Instrument gRPC
        instrumentor = GrpcInstrumentorServer()
        instrumentor.instrument()
        
        logger.info(f"OTLP Tracing initialized successfully. Exporting to {endpoint}")
    except Exception as e:
        logger.warning(f"OpenTelemetry initialization failed: {e}. Distributed tracing is disabled.")
