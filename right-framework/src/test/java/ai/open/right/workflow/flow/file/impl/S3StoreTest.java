package ai.open.right.workflow.flow.file.impl;

import ai.open.right.workflow.flow.WorkflowTask;
import org.easymock.Capture;
import org.easymock.EasyMock;
import org.easymock.IMocksControl;
import org.junit.Assert;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import software.amazon.awssdk.core.sync.RequestBody;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.PutObjectRequest;
import software.amazon.awssdk.services.s3.model.PutObjectResponse;
import software.amazon.awssdk.services.s3.presigner.S3Presigner;
import software.amazon.awssdk.services.s3.presigner.model.GetObjectPresignRequest;
import software.amazon.awssdk.services.s3.presigner.model.PresignedGetObjectRequest;

import java.net.URL;

import static org.easymock.EasyMock.capture;
import static org.easymock.EasyMock.createControl;
import static org.easymock.EasyMock.createMock;
import static org.easymock.EasyMock.expect;
import static org.easymock.EasyMock.expectLastCall;
import static org.easymock.EasyMock.isA;
import static org.easymock.EasyMock.newCapture;
import static org.junit.jupiter.api.Assertions.*;

/**
 * S3Store 单元测试类
 */
class S3StoreTest {

    private S3Store s3Store;
    private S3Client s3Client;
    private S3Presigner s3Presigner;
    private IMocksControl control;

    @BeforeEach
    void setUp() {
        // 创建 EasyMock 控制器和 Mock 对象
        control = createControl();
        s3Client = control.createMock(S3Client.class);
        s3Presigner = control.createMock(S3Presigner.class);

        // 初始化被测对象并注入 Mock
        s3Store = new S3Store();
        s3Store.setClient(s3Client);
        s3Store.setPreSigner(s3Presigner);
        s3Store.setBucket("test-bucket");
        s3Store.setTimeout(300000);
        s3Store.setOversize(10086);
        s3Store.setAccess("test-access");
        s3Store.setSecret("test-secret");
        s3Store.setRegion("ap-southeast-1");
        s3Store.setPrefix("right/");
    }

    @Test
    void testName() throws Exception {
        assertEquals(S3Store.NAME, s3Store.name());
    }

    @Test
    void testSupportNetwork() throws Exception {
        assertTrue(s3Store.supportNetwork());
    }

    @Test
    void testSupportFilesys() throws Exception {
        assertFalse(s3Store.supportFilesys());
    }

    @Test
    void testInit() throws Exception {
        // 测试 init 方法，验证其能正确创建 S3 客户端实例
        S3Store store = new S3Store();
        store.setAccess("access");
        store.setSecret("secret");
        store.setOversize(10086);
        store.setRegion("ap-southeast-1");

        // 执行初始化
        store.init();

        // 验证客户端是否已创建
        assertNotNull(store.getClient());
        assertNotNull(store.getPreSigner());
        Assert.assertEquals(Integer.valueOf(10086), store.getOversize());
        // 清理资源
        store.destroy();
    }

    @Test
    void testDestroy() throws Exception {
        // 录制行为：预期调用 close 方法
        s3Client.close();
        expectLastCall();
        s3Presigner.close();
        expectLastCall();

        control.replay();

        // 执行销毁
        s3Store.destroy();

        // 验证调用
        control.verify();
    }

    @Test
    void testStore() throws Exception {
        byte[] bytes = "test data".getBytes();
        String suffix = "txt";
        WorkflowTask task = null;

        // 1. 录制行为：预期调用 putObject
        expect(s3Client.putObject(isA(PutObjectRequest.class), isA(RequestBody.class)))
                .andReturn(PutObjectResponse.builder().build());

        // 2. PresignedGetObjectRequest 用 Mockito 构造（EasyMock 代理该类型在部分 JVM 会报错）
        URL mockUrl = new URL("https://test-bucket.s3.ap-southeast-1.amazonaws.com/mock-file.txt");
        PresignedGetObjectRequest presignedRequest = Mockito.mock(PresignedGetObjectRequest.class);
        Mockito.when(presignedRequest.url()).thenReturn(mockUrl);
        expect(s3Presigner.presignGetObject(isA(GetObjectPresignRequest.class)))
                .andReturn(presignedRequest);

        control.replay();

        // 执行存储逻辑（内部调用 store(RequestBody, key)，key 为 UUID + suffix）
        String result = s3Store.store(bytes, suffix, task);

        // 验证结果
        assertNotNull(result);
        assertEquals("https://test-bucket.s3.ap-southeast-1.amazonaws.com/mock-file.txt", result);

        control.verify();
    }

    @Test
    void testStoreWithRequestBodyAndKey() throws Exception {
        RequestBody requestBody = RequestBody.fromBytes("hello".getBytes());
        String key = "my-file.txt";
        String expectedPath = "right/my-file.txt";

        Capture<PutObjectRequest> putCapture = newCapture();
        expect(s3Client.putObject(capture(putCapture), isA(RequestBody.class)))
                .andReturn(PutObjectResponse.builder().build());

        URL mockUrl = new URL("https://test-bucket.s3.ap-southeast-1.amazonaws.com/right/my-file.txt");
        PresignedGetObjectRequest presignedRequest = Mockito.mock(PresignedGetObjectRequest.class);
        Mockito.when(presignedRequest.url()).thenReturn(mockUrl);
        expect(s3Presigner.presignGetObject(isA(GetObjectPresignRequest.class)))
                .andReturn(presignedRequest);

        control.replay();

        String result = s3Store.store(requestBody, key);

        assertNotNull(result);
        assertEquals("https://test-bucket.s3.ap-southeast-1.amazonaws.com/right/my-file.txt", result);

        PutObjectRequest captured = putCapture.getValue();
        assertEquals("test-bucket", captured.bucket());
        assertEquals(expectedPath, captured.key());

        control.verify();
    }

    @Test
    void testInitConfig() throws Exception {
        // 测试静态内部配置类 InitConfig
        S3Store.InitConfig config = new S3Store.InitConfig();
        config.setAccess("access-key");
        config.setSecret("secret-key");
        config.setBucket("my-bucket");
        config.setTimeout(600000);
        config.setOversize(10086);
        config.setRegion("us-east-1");

        // 创建 S3Store 实例
        S3Store store = config.s3Store();

        // 验证属性拷贝是否正确
        assertNotNull(store);
        assertEquals(Integer.valueOf(10086), store.getOversize());
        assertEquals("access-key", store.getAccess());
        assertEquals("secret-key", store.getSecret());
        assertEquals("my-bucket", store.getBucket());
        assertEquals(600000, store.getTimeout());
        assertEquals("us-east-1", store.getRegion());
    }
}
