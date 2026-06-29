package team._8.aitrogiang;

import io.github.cdimascio.dotenv.Dotenv;
import io.github.cdimascio.dotenv.DotenvEntry;
import org.springframework.boot.SpringApplication;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
@EnableJpaRepositories(basePackages = "team._8.aitrogiang.repository")
public class AitrogiangApplication {

	public static void main(String[] args) {
		SpringApplication.run(AitrogiangApplication.class, args);
	}

}
