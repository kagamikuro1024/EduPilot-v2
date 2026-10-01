package team._8.aitrogiang.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ReadListener;
import jakarta.servlet.ServletException;
import jakarta.servlet.ServletInputStream;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletRequestWrapper;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.util.StreamUtils;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.regex.Pattern;

@Component
public class PrivacyFirewallFilter extends OncePerRequestFilter {

    private static final Pattern MSSV_PATTERN = Pattern.compile("\\b\\d{8}\\b");
    private static final Pattern PHONE_PATTERN = Pattern.compile("(0|\\+84)\\d{9}");
    private final ObjectMapper objectMapper = new ObjectMapper();
    private static final int MAX_PAYLOAD_SIZE = 102400; // 100KB

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain)
            throws ServletException, IOException {

        String path = request.getRequestURI();
        System.out.println("[DEBUG] PrivacyFirewallFilter: path=" + path + ", method=" + request.getMethod());
        // Target public endpoints for ask-ai
        if (path.contains("/ask-ai") && "POST".equalsIgnoreCase(request.getMethod())) {
            
            long contentLength = request.getContentLengthLong();
            if (contentLength > MAX_PAYLOAD_SIZE) {
                response.sendError(HttpStatus.PAYLOAD_TOO_LARGE.value(), "Payload exceeds 100KB limit");
                return;
            }

            byte[] body;
            try (InputStream is = request.getInputStream()) {
                // Use a bounded read to prevent memory exhaustion
                body = readLimited(is, MAX_PAYLOAD_SIZE);
                
                // If we reached the limit, try to read one more byte to check for overflow
                if (body.length >= MAX_PAYLOAD_SIZE && is.read() != -1) {
                    response.sendError(HttpStatus.PAYLOAD_TOO_LARGE.value(), "Payload exceeds 100KB limit");
                    return;
                }
            }

            String bodyString = new String(body, StandardCharsets.UTF_8);

            if (containsPII(bodyString)) {
                handlePIIDetected(response);
                return;
            }

            // Wrap the request with the cached body so it can be read again downstream
            HttpServletRequest wrappedRequest = new CachedBodyHttpServletRequest(request, body);
            filterChain.doFilter(wrappedRequest, response);
        } else {
            filterChain.doFilter(request, response);
        }
    }

    private byte[] readLimited(InputStream is, int limit) throws IOException {
        java.io.ByteArrayOutputStream baos = new java.io.ByteArrayOutputStream();
        byte[] buffer = new byte[4096];
        int totalRead = 0;
        int bytesRead;
        while (totalRead < limit && (bytesRead = is.read(buffer, 0, Math.min(buffer.length, limit - totalRead))) != -1) {
            baos.write(buffer, 0, bytesRead);
            totalRead += bytesRead;
        }
        return baos.toByteArray();
    }

    private boolean containsPII(String text) {
        return MSSV_PATTERN.matcher(text).find() || PHONE_PATTERN.matcher(text).find();
    }

    private void handlePIIDetected(HttpServletResponse response) throws IOException {
        response.setStatus(HttpStatus.FORBIDDEN.value());
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        response.setCharacterEncoding(StandardCharsets.UTF_8.name());

        Map<String, String> error = Map.of(
                "error_code", "ERR_PII_DETECTED",
                "error", "ERR_PII_DETECTED",
                "message", "Vui lòng chuyển sang khung chat Private để gửi thông tin cá nhân.",
                "suggested_channel", "PRIVATE"
        );

        try (java.io.PrintWriter writer = response.getWriter()) {
            writer.write(objectMapper.writeValueAsString(error));
            writer.flush();
        }
    }

    private static class CachedBodyHttpServletRequest extends HttpServletRequestWrapper {
        private final byte[] cachedBody;

        public CachedBodyHttpServletRequest(HttpServletRequest request, byte[] cachedBody) {
            super(request);
            this.cachedBody = cachedBody;
        }

        @Override
        public ServletInputStream getInputStream() {
            return new CachedBodyServletInputStream(this.cachedBody);
        }

        @Override
        public java.io.BufferedReader getReader() {
            return new java.io.BufferedReader(new java.io.InputStreamReader(new ByteArrayInputStream(this.cachedBody)));
        }
    }

    private static class CachedBodyServletInputStream extends ServletInputStream {
        private final InputStream cachedBodyInputStream;

        public CachedBodyServletInputStream(byte[] cachedBody) {
            this.cachedBodyInputStream = new ByteArrayInputStream(cachedBody);
        }

        @Override
        public boolean isFinished() {
            try {
                return cachedBodyInputStream.available() == 0;
            } catch (IOException e) {
                return true;
            }
        }

        @Override
        public boolean isReady() {
            return true;
        }

        @Override
        public void setReadListener(ReadListener readListener) {
        }

        @Override
        public int read() throws IOException {
            return cachedBodyInputStream.read();
        }
    }
}
