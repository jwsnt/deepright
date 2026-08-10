package ai.open.right.workflow.flow.script;

import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.config.TimeoutConfig;
import org.junit.Assert;
import org.junit.Test;

public class ScriptConfigTest {

    @Test
    public void testTimeOutDefault() {
        TimeoutConfig timeoutConfig = new TimeoutConfig();
        timeoutConfig.setTimeout(123);
        timeoutConfig.setTimeout4Llm(112);
        timeoutConfig.setTimeout4Service(456);
        timeoutConfig.setTimeout4Corrector(789);
        timeoutConfig.setTimeout4Condition(101);
        ScriptConfig scriptConfig = new ScriptConfig();
        Assert.assertFalse(scriptConfig.hasNotifier());
        scriptConfig.setNotifier("NOTIFIER");
        Assert.assertTrue(scriptConfig.hasNotifier());
        scriptConfig.setTimeout(timeoutConfig);
        Assert.assertEquals("NOTIFIER", scriptConfig.getNotifier());
        Assert.assertEquals(timeoutConfig, scriptConfig.getTimeout());
        Assert.assertEquals(Integer.valueOf(789), scriptConfig.getTimeout4Corrector(1));
        Assert.assertEquals(Integer.valueOf(101), scriptConfig.getTimeout4Condition(1));
        Assert.assertEquals(Integer.valueOf(123), scriptConfig.getTimeout(1));
    }

    @Test
    public void testWrapSet() {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setWrap(ScriptConfig.WRAP_OBJECT);
        Assert.assertEquals(ScriptConfig.WRAP_OBJECT, scriptConfig.getWrap());
        Assert.assertTrue(scriptConfig.shouldWrap());
    }

    @Test
    public void testSuccessCode() {
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setSuccessCode(500);
        Assert.assertEquals(ProtocolCode.C500, scriptConfig.getSuccessCode());
        Assert.assertTrue(scriptConfig.isSuccessCode(500));
        Assert.assertFalse(scriptConfig.isSuccessCode(200));
    }

    @Test
    public void testInit() {
        ScriptConfig scriptConfig = new ScriptConfig();
        Assert.assertEquals("NOTIFIER1", scriptConfig.init("NOTIFIER1").getNotifier());
        Assert.assertEquals("NOTIFIER1", scriptConfig.init("NOTIFIER2").getNotifier());
    }

    @Test
    public void testMerge() throws Exception {
        ScriptConfig base = new ScriptConfig();
        ScriptConfig override = new ScriptConfig();
        base.setSuccessCode(200);
        override.setSuccessCode(300);
        base.merge(override);
        Assert.assertEquals(Integer.valueOf(200), base.getSuccessCode());
        base.setSuccessCode(null);
        base.merge(override);
        Assert.assertEquals(Integer.valueOf(300), base.getSuccessCode());
        base.setCondition("baseCondition");
        override.setCondition("overrideCondition");
        base.merge(override);
        Assert.assertEquals("baseCondition", base.getCondition());
        base.setCondition(null);
        base.merge(override);
        Assert.assertEquals("overrideCondition", base.getCondition());
        ScriptCorrector baseCorrector = new ScriptCorrector();
        ScriptCorrector overrideCorrector = new ScriptCorrector();
        base.setCorrector(baseCorrector);
        base.merge(override);
        Assert.assertEquals(baseCorrector, base.getCorrector());
        base.setCorrector(null);
        override.setCorrector(overrideCorrector);
        base.merge(override);
        Assert.assertEquals(overrideCorrector, base.getCorrector());
        base.setNotifier("baseNotifier");
        override.setNotifier("overrideNotifier");
        base.merge(override);
        Assert.assertEquals("baseNotifier", base.getNotifier());
        base.setNotifier(null);
        base.merge(override);
        Assert.assertEquals("overrideNotifier", base.getNotifier());
        base.setEngine("baseEngine");
        override.setEngine("overrideEngine");
        base.merge(override);
        Assert.assertEquals("baseEngine", base.getEngine());
        base.setEngine(null);
        base.merge(override);
        Assert.assertEquals("overrideEngine", base.getEngine());
        TimeoutConfig baseTimeout = new TimeoutConfig();
        TimeoutConfig overrideTimeout = new TimeoutConfig();
        base.setTimeout(baseTimeout);
        base.merge(override);
        Assert.assertEquals(baseTimeout, base.getTimeout());
        base.setTimeout(null);
        override.setTimeout(overrideTimeout);
        base.merge(override);
        Assert.assertEquals(overrideTimeout, base.getTimeout());
        base.setWrap("baseWrap");
        override.setWrap("overrideWrap");
        base.merge(override);
        Assert.assertEquals("baseWrap", base.getWrap());
        base.setWrap(null);
        base.merge(override);
        Assert.assertEquals("overrideWrap", base.getWrap());
        base.merge(null);
        Assert.assertEquals("overrideWrap", base.getWrap());
    }
    @Test
    public void testGetEngineDefault() {
        ScriptConfig config = new ScriptConfig();
        config.setEngine(null);
        Assert.assertEquals(ScriptConfig.ENGINE_PYTHON, config.getEngine());
    }

    @Test
    public void testMergeNull() throws Exception {
        ScriptConfig config = new ScriptConfig();
        config.setEngine("E");
        Assert.assertEquals("E", config.merge(null).getEngine());
    }
}
