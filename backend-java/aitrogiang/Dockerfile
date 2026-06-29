# Build stage
FROM eclipse-temurin:21-jdk-jammy AS builder
WORKDIR /app

# Copy gradle wrapper and config files
COPY backend-java/aitrogiang/gradlew backend-java/aitrogiang/
COPY backend-java/aitrogiang/gradle backend-java/aitrogiang/gradle
COPY backend-java/aitrogiang/build.gradle backend-java/aitrogiang/settings.gradle backend-java/aitrogiang/
COPY shared-proto shared-proto
WORKDIR /app/backend-java/aitrogiang

# Make gradlew executable
RUN chmod +x gradlew

# Download dependencies (this layer is cached until build.gradle changes)
RUN ./gradlew dependencies --no-daemon || true

# Copy the rest of the source code
COPY backend-java/aitrogiang/src src

# Build the application
RUN ./gradlew bootJar -x test --no-daemon

# Run stage
FROM eclipse-temurin:21-jre-jammy
WORKDIR /app

# Copy built artifact from builder stage
COPY --from=builder /app/backend-java/aitrogiang/build/libs/*.jar app.jar

EXPOSE 8080
ENTRYPOINT ["java", "-Xms256m", "-Xmx512m", "-jar", "app.jar"]
