package ai.open.right.workflow.flow.script;

import org.junit.Assert;
import org.junit.Test;

public class ScriptResponseTest {

    @Test
    public void test1() {
        ScriptResponse scriptResponse = ScriptResponse.builder().build();
        scriptResponse.setData("HELLO");
        scriptResponse.setCode(200);
        Assert.assertEquals("HELLO",scriptResponse.getData());
        Assert.assertEquals(Integer.valueOf(200),scriptResponse.getCode());
    }

    @Test
    public void test2() {
        ScriptResponse scriptResponse = new ScriptResponse();
        scriptResponse.setData("HELLO");
        scriptResponse.setCode(200);
        Assert.assertEquals("HELLO",scriptResponse.getData());
        Assert.assertEquals(Integer.valueOf(200),scriptResponse.getCode());
    }
}
