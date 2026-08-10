package ai.open.right.workflow.flow.file.impl;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

/**
 * S3StoreInitConfig 单元测试
 * 验证手动创建 InitConfig 并生成 S3Store Bean 的逻辑
 */
class S3StoreInitConfigTest {

    @Test
    @DisplayName("测试 S3Store Bean 的创建及属性拷贝逻辑")
    void testS3StoreBeanCreation() throws Exception {
        // 1. 准备测试数据
        Integer timeout = 60000;
        String bucket = "test-bucket";
        String access = "test-access-key";
        String secret = "test-secret-key";
        String region = "cn-north-1";

        // 2. 手动创建 InitConfig 并设置属性
        S3Store.InitConfig initConfig = new S3Store.InitConfig();
        initConfig.setTimeout(timeout);
        initConfig.setBucket(bucket);
        initConfig.setAccess(access);
        initConfig.setSecret(secret);
        initConfig.setRegion(region);

        // 3. 执行 Bean 创建方法
        S3Store s3Store = initConfig.s3Store();

        // 4. 验证属性是否正确拷贝到 S3Store 实例中
        Assertions.assertNotNull(s3Store, "生成的 S3Store 实例不应为空");
        Assertions.assertEquals(timeout, s3Store.getTimeout(), "Timeout 属性不匹配");
        Assertions.assertEquals(bucket, s3Store.getBucket(), "Bucket 属性不匹配");
        Assertions.assertEquals(access, s3Store.getAccess(), "Access 属性不匹配");
        Assertions.assertEquals(secret, s3Store.getSecret(), "Secret 属性不匹配");
        Assertions.assertEquals(region, s3Store.getRegion(), "Region 属性不匹配");
    }

    @Test
    @DisplayName("测试 S3Store Bean 创建时的默认值或空值处理")
    void testS3StoreBeanCreationWithEmptyValues() throws Exception {
        // 1. 创建空的 InitConfig
        S3Store.InitConfig initConfig = new S3Store.InitConfig();

        // 2. 执行 Bean 创建方法
        S3Store s3Store = initConfig.s3Store();

        // 3. 验证实例存在且属性为 null (因为手动创建时没有 Spring 的 @Value 注入默认值)
        Assertions.assertNotNull(s3Store);
        Assertions.assertNull(s3Store.getBucket());
        Assertions.assertNull(s3Store.getAccess());
    }
}
