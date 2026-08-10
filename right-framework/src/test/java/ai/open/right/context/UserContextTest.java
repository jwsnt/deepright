package ai.open.right.context;

import com.fasterxml.jackson.core.JsonParseException;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashMap;
import java.util.Map;

public class UserContextTest {

    @Test
    public void testCheckAllPass() {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setSystem("S1");
        userContext.setRegion("S2");
        userContext.setModel("S3");
        userContext.setBrand("S5");
        userContext.setDevice("S6");
        Assert.assertEquals("S4", userContext.getLanguage());
        Assert.assertEquals("S1", userContext.getSystem());
        Assert.assertEquals("S2", userContext.getRegion());
        Assert.assertEquals("S3", userContext.getModel());
        Assert.assertEquals("S5", userContext.getBrand());
        Assert.assertEquals("S6", userContext.getDevice());
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutLa() throws Exception {
        UserContext userContext = UserContext.builder().build();
        userContext.setSystem("S1");
        userContext.setRegion("S2");
        userContext.setModel("S3");
        userContext.setBrand("S5");
        userContext.setDevice("S6");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutSy() throws Exception {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setRegion("S2");
        userContext.setModel("S3");
        userContext.setBrand("S5");
        userContext.setDevice("S6");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutRe() {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setSystem("S1");
        userContext.setModel("S3");
        userContext.setBrand("S5");
        userContext.setDevice("S6");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutMo() {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setSystem("S1");
        userContext.setRegion("S2");
        userContext.setBrand("S5");
        userContext.setDevice("S6");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutBr() {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setSystem("S1");
        userContext.setRegion("S2");
        userContext.setModel("S3");
        userContext.setDevice("S6");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCheckWithoutDe() {
        UserContext userContext = UserContext.builder().build();
        userContext.setLanguage("S4");
        userContext.setSystem("S1");
        userContext.setRegion("S2");
        userContext.setModel("S3");
        userContext.setBrand("S5");
        UserContext.UserContextChecker.check(userContext);
    }

    @Test
    public void testSetDefaultWithNull() {
        // 测试传入 null 时是否返回带有默认值的对象
        UserContext context = UserContext.setDefault(null);
        Assert.assertNotNull(context);
        Assert.assertEquals(UserContext.UNKNOWN, context.getLanguage());
        Assert.assertEquals(UserContext.UNKNOWN, context.getSystem());
        Assert.assertEquals(UserContext.UNKNOWN, context.getRegion());
        Assert.assertEquals(UserContext.UNKNOWN, context.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, context.getBrand());
        Assert.assertEquals(UserContext.UNKNOWN, context.getDevice());
    }

    @Test
    public void testSetDefaultWithEmptyFields() {
        // 测试传入空字段的对象时，字段是否被填充为 UNKNOWN
        UserContext context = new UserContext();
        UserContext result = UserContext.setDefault(context);
        Assert.assertSame(context, result);
        Assert.assertEquals(UserContext.UNKNOWN, result.getLanguage());
        Assert.assertEquals(UserContext.UNKNOWN, result.getSystem());
        Assert.assertEquals(UserContext.UNKNOWN, result.getRegion());
        Assert.assertEquals(UserContext.UNKNOWN, result.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, result.getBrand());
        Assert.assertEquals(UserContext.UNKNOWN, result.getDevice());
    }

    @Test
    public void testSetDefaultWithPartialFields() {
        // 测试部分字段有值时，其他空字段是否被填充
        UserContext context = new UserContext();
        context.setLanguage("zh");
        context.setSystem("iOS");
        UserContext result = UserContext.setDefault(context);
        Assert.assertEquals("zh", result.getLanguage());
        Assert.assertEquals("iOS", result.getSystem());
        Assert.assertEquals(UserContext.UNKNOWN, result.getRegion());
        Assert.assertEquals(UserContext.UNKNOWN, result.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, result.getBrand());
        Assert.assertEquals(UserContext.UNKNOWN, result.getDevice());
    }

    @Test
    public void testToString() {
        // 测试 toString 方法
        UserContext context = UserContext.builder()
                .language("en")
                .system("Android")
                .build();
        String toString = context.toString();
        Assert.assertTrue(toString.contains("language=en"));
        Assert.assertTrue(toString.contains("system=Android"));
    }

    @Test
    public void testNoArgsConstructor() {
        // 测试无参构造函数
        UserContext context = new UserContext();
        Assert.assertNull(context.getLanguage());
        Assert.assertNull(context.getSystem());
    }

    @org.junit.jupiter.api.Test
    public void testBuilder() {
        UserContext context = UserContext.builder()
                .language("en")
                .device("iphone")
                .build();
        org.junit.jupiter.api.Assertions.assertEquals("en", context.getLanguage());
        org.junit.jupiter.api.Assertions.assertEquals("iphone", context.getDevice());
    }

    @org.junit.jupiter.api.Test
    public void testSetDefaultWithToken() {
        UserContext context = UserContext.builder().token("my-token").build();
        UserContext result = UserContext.setDefault(context);
        org.junit.jupiter.api.Assertions.assertEquals("my-token", result.getToken());
        org.junit.jupiter.api.Assertions.assertEquals(UserContext.UNKNOWN, result.getLanguage());
    }

    @Test
    public void testGetMetadataWhenMetadataNull() {
        UserContext context = new UserContext();
        Assert.assertNull(context.getMetadata("anyKey"));
    }

    @Test
    public void testPutMetadataInitializesMapAndStoresValue() {
        UserContext context = new UserContext();
        context.putMetadata("k1", "v1");
        Assert.assertEquals("v1", context.getMetadata("k1"));
        Assert.assertNotNull(context.getMetadata());
        Assert.assertEquals(1, context.getMetadata().size());
    }

    @Test
    public void testPutMetadataAppendsToExistingMap() {
        Map<String, Object> existing = new HashMap<>();
        existing.put("a", 1);
        UserContext context = UserContext.builder().build();
        context.putMetadata("b", 2);
        Assert.assertEquals(null, context.getMetadata("a"));
        Assert.assertEquals(Integer.valueOf(2), context.getMetadata("b"));
    }

    @Test
    public void testPutMetadataOverwritesSameKey() {
        UserContext context = new UserContext();
        context.putMetadata("k", "first");
        context.putMetadata("k", "second");
        Assert.assertEquals("second", context.getMetadata("k"));
    }

    @Test
    public void testGetMetadataMissingKeyReturnsNull() {
        UserContext context = new UserContext();
        context.putMetadata("only", "x");
        Assert.assertNull(context.getMetadata("missing"));
    }

    @Test
    public void testSetDefaultPreservesMetadata() {
        UserContext context = new UserContext();
        context.putMetadata("traceId", "abc-123");
        UserContext result = UserContext.setDefault(context);
        Assert.assertSame(context, result);
        Assert.assertEquals("abc-123", result.getMetadata("traceId"));
    }

    @Test
    public void getMetadata_withClazz_returnsTypedValue() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("n", 42);
        context.putMetadata("s", "hello");
        Assert.assertEquals(Integer.valueOf(42), context.getMetadata("n", Integer.class));
        Assert.assertEquals("hello", context.getMetadata("s", String.class));
    }

    @Test
    public void getMetadata_withClazz_metadataNull_returnsNull() throws Exception {
        UserContext context = new UserContext();
        Assert.assertNull(context.getMetadata("any", String.class));
    }

    @Test
    public void getMetadata_withClazz_missingKey_returnsNull() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("k", "v");
        Assert.assertNull(context.getMetadata("missing", String.class));
    }

    @Test(expected = JsonParseException.class)
    public void getMetadata_withClazz_wrongType_throwsClassCastException() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("k", "not-a-number");
        context.getMetadata("k", Integer.class);
    }

    @Test
    public void delMetadata_withClazz_returnsRemovedTypedValueAndDeletesEntry() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("k", "v");
        context.putMetadata("other", 1);

        Assert.assertEquals("v", context.delMetadata("k", String.class));
        Assert.assertNull(context.getMetadata("k"));
        Assert.assertEquals(Integer.valueOf(1), context.getMetadata("other", Integer.class));
    }

    @Test
    public void delMetadata_withClazz_missingKey_returnsNull() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("k", "v");

        Assert.assertNull(context.delMetadata("missing", String.class));
        Assert.assertEquals("v", context.getMetadata("k", String.class));
    }

    @Test(expected = ClassCastException.class)
    public void delMetadata_withClazz_wrongType_throwsClassCastException() throws Exception {
        UserContext context = new UserContext();
        context.putMetadata("k", "v");

        context.delMetadata("k", Integer.class);
    }

    @Test
    public void testCopyWithDeviceOverridesDeviceAndKeepsSourceUnchanged() {
        UserContext context = UserContext.builder()
                .language("zh")
                .system("iOS")
                .device("iphone")
                .region("CN")
                .brand("Apple")
                .model("iPhone 16")
                .token("token")
                .build();

        UserContext copied = UserContext.copyWithDevice(context, "ipad");

        Assert.assertNotSame(context, copied);
        Assert.assertEquals("zh", copied.getLanguage());
        Assert.assertEquals("ipad", copied.getDevice());
        Assert.assertEquals("CN", copied.getRegion());
        Assert.assertEquals("Apple", copied.getBrand());
        Assert.assertEquals("iPhone 16", copied.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, copied.getSystem());
        Assert.assertNull(copied.getToken());
        Assert.assertEquals("iphone", context.getDevice());
        Assert.assertEquals("iOS", context.getSystem());
        Assert.assertEquals("token", context.getToken());
    }

    @Test
    public void testCopyWithDeviceFallsBackToOriginalDeviceWhenDeviceIsNull() {
        UserContext context = UserContext.builder()
                .language("en")
                .device("android")
                .region("US")
                .brand("Google")
                .model("Pixel")
                .build();

        UserContext copied = UserContext.copyWithDevice(context, null);

        Assert.assertEquals("en", copied.getLanguage());
        Assert.assertEquals("android", copied.getDevice());
        Assert.assertEquals("US", copied.getRegion());
        Assert.assertEquals("Google", copied.getBrand());
        Assert.assertEquals("Pixel", copied.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, copied.getSystem());
    }

    @Test
    public void testCopyCreatesNewContextAndAppliesDefaultsForMissingFields() {
        UserContext context = UserContext.builder()
                .language("ja")
                .device("tablet")
                .region("JP")
                .brand("Sony")
                .model("Xperia")
                .system("Android")
                .token("secret")
                .build();

        UserContext copied = UserContext.copy(context);

        Assert.assertNotSame(context, copied);
        Assert.assertEquals("ja", copied.getLanguage());
        Assert.assertEquals("tablet", copied.getDevice());
        Assert.assertEquals("JP", copied.getRegion());
        Assert.assertEquals("Sony", copied.getBrand());
        Assert.assertEquals("Xperia", copied.getModel());
        Assert.assertEquals(UserContext.UNKNOWN, copied.getSystem());
        Assert.assertNull(copied.getToken());
        Assert.assertEquals("Android", context.getSystem());
        Assert.assertEquals("secret", context.getToken());
    }

}
