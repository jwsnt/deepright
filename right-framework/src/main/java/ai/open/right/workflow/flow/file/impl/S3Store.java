package ai.open.right.workflow.flow.file.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.core.sync.RequestBody;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.PutObjectRequest;
import software.amazon.awssdk.services.s3.presigner.S3Presigner;
import software.amazon.awssdk.services.s3.presigner.model.GetObjectPresignRequest;

import java.time.Duration;
import java.util.UUID;

@Slf4j
@Getter
@Setter
public class S3Store extends SafeStore {

    public static final String NAME = "file.store.s3";

    protected S3Presigner preSigner;

    protected S3Client client;

    protected Integer timeout;

    protected String bucket;

    protected String access;

    protected String secret;

    protected String region;

    protected String prefix;

    @PostConstruct
    public void init() throws Exception {
        StaticCredentialsProvider provider = StaticCredentialsProvider.create(AwsBasicCredentials.create(this.access, this.secret));
        Region region = Region.of(this.region);
        this.preSigner = S3Presigner.builder()
                .region(region)
                .credentialsProvider(provider)
                .build();
        this.client = S3Client.builder()
                .credentialsProvider(provider)
                .region(region)
                .build();
    }

    @PreDestroy
    public void destroy() throws Exception {
        this.preSigner.close();
        this.client.close();
    }

    @Override
    public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
        this.check(bytes);
        return this.store(bytes, suffix);
    }

    @Override
    public String store(byte[] bytes, String suffix) throws Exception {
        this.check(bytes);
        return this.store(RequestBody.fromBytes(bytes), UUID.randomUUID() + (!StringUtils.isEmpty(suffix) ? (StringUtils.startsWithIgnoreCase(suffix, ".") ? suffix : ("." + suffix)) : ""));
    }

    @Override
    public Boolean supportNetwork() throws Exception {
        return true;
    }

    @Override
    public Boolean supportFilesys() throws Exception {
        return false;
    }

    @Override
    public String name() throws Exception {
        return S3Store.NAME;
    }

    protected String store(RequestBody requestBody, String key) throws Exception {
        String path = this.buildKey(key);
        this.client.putObject(PutObjectRequest.builder()
                .key(path)
                .bucket(this.bucket)
                .build(), requestBody);
        return this.buildPresign(path);
    }

    protected String buildPresign(String key) throws Exception {
        String path = this.preSigner.presignGetObject(GetObjectPresignRequest.builder()
                .getObjectRequest(r -> r.bucket(this.bucket).key(key).build())
                .signatureDuration(Duration.ofMillis(this.timeout))
                .build()).url().toString();
        if (log.isDebugEnabled()) {
            log.debug("The s3 upload path={}", path);
        }
        return path;
    }

    protected String buildKey(String key) throws Exception {
        return this.prefix + key;
    }

    @ConditionalOnProperty(name = "file.store.s3.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Value("${file.store.s3.timeout:300000}")
        protected Integer timeout;

        @Value("${file.store.s3.bucket:}")
        protected String bucket;

        @Value("${file.store.s3.access:}")
        protected String access;

        @Value("${file.store.s3.secret:}")
        protected String secret;

        @Value("${file.store.s3.region:ap-southeast-1}")
        protected String region;

        @Value("${file.store.s3.prefix:right}")
        protected String prefix;

        @Bean(S3Store.NAME)
        @ConditionalOnMissingBean(name = S3Store.NAME)
        public S3Store s3Store() throws Exception {
            S3Store s3Store = new S3Store();
            BeanUtils.copyProperties(this, s3Store);
            log.info("S3Store inited: timeout={}, bucket_length={}, access_length={}, secret_length={}, region={}", s3Store.getTimeout(), StringUtils.length(s3Store.getBucket()), StringUtils.length(s3Store.getAccess()), StringUtils.length(s3Store.getSecret()), s3Store.getRegion());
            return s3Store;
        }
    }
}
