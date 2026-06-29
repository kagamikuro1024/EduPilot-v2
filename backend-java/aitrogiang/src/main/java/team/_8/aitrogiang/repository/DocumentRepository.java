package team._8.aitrogiang.repository;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import team._8.aitrogiang.model.Document;
import team._8.aitrogiang.model.DocumentStatus;

import java.util.List;
import java.util.UUID;

@Repository
public interface DocumentRepository extends JpaRepository<Document, UUID> {
    List<Document> findByUploadedById(UUID uploadedById);
    long countByStatus(DocumentStatus status);

    @Modifying
    @Query(value = "DELETE FROM document_chunks WHERE document_id = :documentId", nativeQuery = true)
    void deleteChunksByDocumentId(@Param("documentId") UUID documentId);
}
