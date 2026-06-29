package team._8.aitrogiang.config;

import io.grpc.ClientInterceptor;
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.instrumentation.grpc.v1_6.GrpcTelemetry;
import net.devh.boot.grpc.client.interceptor.GrpcGlobalClientInterceptor;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class TracingConfig {

    @Bean
    public OpenTelemetry openTelemetry() {
        return io.opentelemetry.api.GlobalOpenTelemetry.get();
    }

    @Bean
    @GrpcGlobalClientInterceptor
    public ClientInterceptor globalClientInterceptor(OpenTelemetry openTelemetry) {
        return GrpcTelemetry.create(openTelemetry).newClientInterceptor();
    }
}
