package ai.open.right.workflow.flow.llm;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCallRequest;
import com.google.common.collect.ImmutableMap;
import org.junit.Assert;
import org.junit.Test;

import java.util.Date;
import java.util.HashMap;
import java.util.Map;

public class ProviderFunRequestTest {

    @Test
    public void test() {
        Map<String, Object> configs = new HashMap<>();
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder()
                .args(configs)
                .name("NAME")
                .build();
        Assert.assertEquals(configs, providerFunRequest.getArgs());
        Assert.assertEquals("NAME", providerFunRequest.getName());
    }

    @Test
    public void testArgsWithNull() throws Exception {
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder().build();
        Assert.assertNull(JsonUtils.write(providerFunRequest.getArgs()));
    }

    @Test
    public void testArgsWithObject() throws Exception {
        Date date = new Date(1759401334118L);
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder()
                .args(date)
                .build();
        Assert.assertEquals("1759401334118", JsonUtils.write(providerFunRequest.getArgs()));
    }

    @Test
    public void testArgsWithMap() throws Exception {
        ImmutableMap map = ImmutableMap.of("A", "B");
        ProviderFunCallRequest providerFunRequest = ProviderFunCallRequest.builder()
                .args(map)
                .build();
        Assert.assertEquals("{\"A\":\"B\"}", JsonUtils.write(providerFunRequest.getArgs()));
    }

    @Test
    public void testGetterAndSetterForRefer() {
        // 测试 refer 字段的 get/set
        ProviderFunCallRequest request = ProviderFunCallRequest.builder().build();
        Map testRefer = new HashMap();
        // 调用 setter 设置值
        request.setRefer(testRefer);
        // 调用 getter 验证值
        Assert.assertEquals("refer 字段 get/set 验证失败", testRefer, request.getRefer());
    }

    @Test
    public void testGetterAndSetterForArgs() {
        // 测试 args 字段的 get/set
        ProviderFunCallRequest request = ProviderFunCallRequest.builder().build();
        Map<String, Object> testArgs = ImmutableMap.of("key", "value");

        request.setArgs(testArgs);
        Assert.assertEquals("args 字段 get/set 验证失败", testArgs, request.getArgs());
    }

    @Test
    public void testGetterAndSetterForName() {
        // 测试 name 字段的 get/set
        ProviderFunCallRequest request = ProviderFunCallRequest.builder().build();
        String testName = "TEST_SET_NAME";

        request.setName(testName);
        Assert.assertEquals("name 字段 get/set 验证失败", testName, request.getName());
    }

    @Test
    public void testGetterAndSetterForId() {
        // 测试 id 字段的 get/set
        ProviderFunCallRequest request = ProviderFunCallRequest.builder().build();
        String testId = "ID_123456";

        request.setId(testId);
        Assert.assertEquals("id 字段 get/set 验证失败", testId, request.getId());
    }

    @Test
    public void testAllSettersAndGettersTogether() {
        // 一次性测试所有字段的 get/set
        ProviderFunCallRequest request = ProviderFunCallRequest.builder().build();
        // 准备测试数据
        Map<String, Object> refer = new HashMap<>();
        Map<String, Object> args = new HashMap<>();
        args.put("test", "data");
        String name = "COMPLEX_TEST_NAME";
        String id = "COMPLEX_ID_789";
        // 批量设置值
        request.setRefer(refer);
        request.setArgs(args);
        request.setName(name);
        request.setId(id);
        // 批量验证值
        Assert.assertEquals("refer 验证失败", refer, request.getRefer());
        Assert.assertEquals("args 验证失败", args, request.getArgs());
        Assert.assertEquals("name 验证失败", name, request.getName());
        Assert.assertEquals("id 验证失败", id, request.getId());
    }

    @Test
    public void testSetStringAndNull() {
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder().build();
        providerFunCallRequest.setNameIfAbsent("NAME1");
        Assert.assertEquals("NAME1", providerFunCallRequest.getName());
        providerFunCallRequest.setNameIfAbsent("NAME2");
        Assert.assertEquals("NAME1", providerFunCallRequest.getName());
        providerFunCallRequest.setIdIfAbsent("ID1");
        Assert.assertEquals("ID1", providerFunCallRequest.getId());
        providerFunCallRequest.setIdIfAbsent("ID2");
        Assert.assertEquals("ID1", providerFunCallRequest.getId());
        providerFunCallRequest.appendArgs(null);
        providerFunCallRequest.appendArgs("HELLO");
        providerFunCallRequest.appendArgs(null);
        Assert.assertEquals("HELLO", providerFunCallRequest.getArgs());
        providerFunCallRequest.appendArgs("WORLD");
        Assert.assertEquals("HELLOWORLD", providerFunCallRequest.getArgs());
    }

    @Test
    public void testSetMap() throws Exception {
        ProviderFunCallRequest providerFunCallRequest = ProviderFunCallRequest.builder().build();
        Map<String, Object> map1 = new HashMap<>();
        map1.put("A", "B");
        providerFunCallRequest.appendArgs(map1);
        Map<String, Object> map2 = new HashMap<>();
        map2.put("A", "B");
        map2.put("C", "D");
        providerFunCallRequest.appendArgs(map2);
        Assert.assertEquals("{\"A\":\"B\",\"C\":\"D\"}", JsonUtils.write(providerFunCallRequest.getArgs()));
    }
}
