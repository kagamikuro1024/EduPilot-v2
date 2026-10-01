package team._8.aitrogiang.model;

import jakarta.persistence.*;
import lombok.*;

import java.util.UUID;

@Entity
@Table(
        name = "forum_thread_tags",
        uniqueConstraints = @UniqueConstraint(name = "unique_thread_tag", columnNames = {"thread_id", "tag_id"})
)
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ForumThreadTag {

    @Id
    @GeneratedValue(strategy = GenerationType.AUTO)
    private UUID id;

    @Column(name = "thread_id")
    private UUID threadId;

    @Column(name = "tag_id")
    private UUID tagId;
}
