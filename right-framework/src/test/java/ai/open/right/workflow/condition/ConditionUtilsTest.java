package ai.open.right.workflow.condition;

import ai.open.right.utils.JsonUtils;
import org.junit.Assert;
import org.junit.Test;
import java.util.HashMap;
import java.util.Map;

public class ConditionUtilsTest {

    @Test
    public void test1() throws Exception {
        Assert.assertFalse(ConditionUtils.checkCondition(null).getCondition());
    }

    @Test
    public void test2() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, "true");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test3() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, "true");
        data.put("HELLO", "WORLD");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test4() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, true);
        data.put("HELLO", "WORLD");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test5() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, 1);
        data.put("HELLO", "WORLD");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test6() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, "1");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test7() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, "yes");
        Assert.assertTrue(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }

    @Test
    public void test8() throws Exception {
        Map<String, Object> data = new HashMap<>();
        data.put(ConditionUtils.KEY, "no");
        Assert.assertFalse(ConditionUtils.checkCondition(JsonUtils.write(data)).getCondition());
    }
}
