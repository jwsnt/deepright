package ai.open.right.workflow.flow.script.impl;

import ai.open.right.ObjectBuilder;
import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.script.ScriptConfig;
import ai.open.right.workflow.flow.script.ScriptCorrector;
import org.junit.Assert;
import org.junit.Test;

import java.util.concurrent.atomic.AtomicInteger;

public class AbstractScriptServiceTest {

    @Test
    public void testWithOutConfig1() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                return "Hello";
            }
        };
        abstractScriptService.setTimeout(1000);
        ScriptConfig scriptConfig = new ScriptConfig();
        Assert.assertEquals("Hello", abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testWithOutConfig2() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                return "Hello";
            }
        };
        abstractScriptService.setTimeout(1000);
        Assert.assertEquals("Hello", abstractScriptService.run(null, ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testWithCondition() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setCondition("Condition");
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("true"));
        Assert.assertEquals("Hello", abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test(expected = WorkflowException.class)
    public void testWithConditionFailed() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        scriptConfig.setCondition("Condition");
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        Assert.assertEquals("Hello", abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test
    public void testWithCorrector() throws Exception {
        AtomicInteger count = new AtomicInteger();
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                if (count.getAndIncrement() == 0) {
                    throw new WorkflowException("", ProtocolCode.C500);
                }
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        Assert.assertEquals("Hello", abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask()));
    }

    @Test(expected = WorkflowException.class)
    public void testWithCorrectorWithOtherException1() throws Exception {
        AtomicInteger count = new AtomicInteger();
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                if (count.getAndIncrement() == 0) {
                    throw new WorkflowException("", ProtocolCode.C503);
                }
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = RuntimeException.class)
    public void testWithCorrectorWithOtherException2() throws Exception {
        AtomicInteger count = new AtomicInteger();
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                if (count.getAndIncrement() == 0) {
                    throw new RuntimeException("");
                }
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = RuntimeException.class)
    public void testWithCorrectorWithOtherException3() throws Exception {
        AtomicInteger count = new AtomicInteger();
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                int code = count.getAndIncrement();
                if (code == 0) {
                    throw new WorkflowException("", ProtocolCode.C500);
                }
                if (code == 1) {
                    throw new RuntimeException("");
                }
                return "Hello";
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = WorkflowException.class)
    public void testWithCorrectorWithOtherException4() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                throw new WorkflowException("", ProtocolCode.C500);
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask());
    }

    @Test(expected = WorkflowException.class)
    public void testWithCorrectorWithAlwaysException() throws Exception {
        AbstractScriptService abstractScriptService = new AbstractScriptService() {
            @Override
            public String run(ScriptEnv scriptEnv, String script, Integer timeout) throws Exception {
                throw new WorkflowException("", ProtocolCode.C503);
            }
        };
        ScriptConfig scriptConfig = new ScriptConfig();
        ScriptCorrector corrector = new ScriptCorrector();
        corrector.setCorrection("Correction");
        scriptConfig.setCorrector(corrector);
        abstractScriptService.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent("false"));
        abstractScriptService.run(scriptConfig, ObjectBuilder.buildWorkflowTask());
    }
    @Test(expected = IllegalArgumentException.class)
    public void testRunEmptyCondition() throws Exception {
        AbstractScriptService service = new AbstractScriptService() {
            @Override public String run(ScriptEnv e, String s, Integer t) { return "H"; }
        };
        service.setNotifierService(ObjectBuilder.buildActualNotifierManagerWithWriteBackContent(""));
        ScriptConfig config = new ScriptConfig();
        config.setCondition("C");
        service.run(config, ObjectBuilder.buildWorkflowTask());
    }
}
