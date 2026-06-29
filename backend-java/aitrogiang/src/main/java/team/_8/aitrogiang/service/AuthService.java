package team._8.aitrogiang.service;

import lombok.RequiredArgsConstructor;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.util.Map;

import team._8.aitrogiang.dto.AuthRequest;
import team._8.aitrogiang.dto.AuthResponse;
import team._8.aitrogiang.dto.RegisterRequest;
import team._8.aitrogiang.model.User;
import team._8.aitrogiang.repository.UserRepository;
import team._8.aitrogiang.exception.UserAlreadyExistsException;
import team._8.aitrogiang.security.JwtService;

@Service
@RequiredArgsConstructor
public class AuthService {
    private final UserRepository repository;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;
    private final AuthenticationManager authenticationManager;

    public AuthResponse register(RegisterRequest request) {
        if (repository.findByEmail(request.getEmail()).isPresent()) {
            throw new UserAlreadyExistsException("Email already registered. Please sign in instead.");
        }
        var user = User.builder()
                .fullName(request.getFullName())
                .email(request.getEmail())
                .password(passwordEncoder.encode(request.getPassword()))
                .role(request.getRole())
                .build();
        repository.save(user);
        var jwtToken = jwtService.generateToken(Map.of("role", user.getRole().name()), user);
        return AuthResponse.builder()
                .token(jwtToken)
                .role(user.getRole())
                .fullName(user.getFullName())
                .build();
    }

    public AuthResponse authenticate(AuthRequest request) {
        System.out.println("[DEBUG] Authenticating user: " + request.getEmail());
        try {
            authenticationManager.authenticate(
                    new UsernamePasswordAuthenticationToken(request.getEmail(), request.getPassword())
            );
            System.out.println("[DEBUG] Authentication successful for: " + request.getEmail());
        } catch (Exception e) {
            System.err.println("[ERROR] Authentication failed: " + e.getMessage());
            throw e;
        }

        var user = repository.findByEmail(request.getEmail()).orElseThrow();
        System.out.println("[DEBUG] User found in DB: " + user.getEmail() + " with role: " + user.getRole());

        try {
            var jwtToken = jwtService.generateToken(Map.of("role", user.getRole().name()), user);
            System.out.println("[DEBUG] JWT Token generated successfully");
            return AuthResponse.builder()
                    .token(jwtToken)
                    .role(user.getRole())
                    .fullName(user.getFullName())
                    .build();
        } catch (Exception e) {
            System.err.println("[ERROR] JWT Generation failed: " + e.getMessage());
            e.printStackTrace();
            throw e;
        }
    }
}
