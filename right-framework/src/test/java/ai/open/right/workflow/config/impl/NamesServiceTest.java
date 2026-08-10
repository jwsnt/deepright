package ai.open.right.workflow.config.impl;

import ai.open.right.WorkflowException;
import org.junit.Assert;
import org.junit.Test;

public class NamesServiceTest {

    @Test
    public void test() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        Assert.assertTrue(mcpNameService.isPrefixTools(NamesServiceImpl.PREFIX_TOOLS + "HELLO"));
        Assert.assertTrue(mcpNameService.isPrefixPrompt(NamesServiceImpl.PREFIX_PROMPT + "HELLO"));
        Assert.assertTrue(mcpNameService.isPrefixWorkflow(NamesServiceImpl.PREFIX_WORKFLOW + "HELLO"));
    }

    @Test
    public void testWithNotEncode() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setEncode(false);
        Assert.assertEquals("OK", mcpNameService.encode("OK"));
    }

    @Test
    public void testDecode1() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        Assert.assertEquals("OK1", mcpNameService.decode(mcpNameService.encode(NamesServiceImpl.PREFIX_WORKFLOW, "OK1", ""))[0]);
        Assert.assertEquals("OK1", mcpNameService.decode(mcpNameService.encode(NamesServiceImpl.PREFIX_RESOURCE, "OK1", ""))[0]);
        Assert.assertEquals("OK1", mcpNameService.decode(mcpNameService.encode(NamesServiceImpl.PREFIX_PROMPT, "OK1", ""))[0]);
        Assert.assertEquals("OK1", mcpNameService.decode(mcpNameService.encode(NamesServiceImpl.PREFIX_TOOLS, "OK1", ""))[0]);
        Assert.assertEquals(1, mcpNameService.decode(mcpNameService.encode(NamesServiceImpl.PREFIX_TOOLS, "OK1", "")).length);
    }

    @Test(expected = WorkflowException.class)
    public void testDecode2() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        // 强制使用Tools解析，并解析错误
        mcpNameService.decode("OK");
    }

    @Test
    public void testCircleDecode() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        String[] name = null;
        String encode = mcpNameService.encode(NamesServiceImpl.PREFIX_PROMPT, "HELLO", "WORLD");
        for (int i = 0; i < 100000; i++) {
            if (name == null) {
                name = mcpNameService.decode(encode);
            } else {
                String[] target = mcpNameService.decode(encode);
                Assert.assertEquals(name[0], target[0]);
                Assert.assertEquals(name[1], target[1]);
            }
        }
    }

    @Test
    public void testCircleEncode() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        String name = null;
        for (int i = 0; i < 100000; i++) {
            if (name == null) {
                name = mcpNameService.encode(NamesServiceImpl.PREFIX_PROMPT, "HELLO", "WORLD");
            } else {
                String target = mcpNameService.encode(NamesServiceImpl.PREFIX_PROMPT, "HELLO", "WORLD");
                Assert.assertEquals(name, target);
            }
        }
    }

    @Test
    public void testCircleEncode2() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        String name = null;
        for (int i = 0; i < 100000; i++) {
            if (name == null) {
                name = mcpNameService.encode(NamesServiceImpl.PREFIX_WORKFLOW, "search", "search");
            } else {
                String target = mcpNameService.encode(NamesServiceImpl.PREFIX_WORKFLOW, "search", "search");
                Assert.assertEquals(name, target);
            }
        }
    }

    @Test
    public void testCircleEncode3() throws Exception {
        NamesServiceImpl mcpNameService = new NamesServiceImpl();
        mcpNameService.init();
        mcpNameService.setLength(8);
        String name = null;
        for (int i = 0; i < 100000; i++) {
            if (name == null) {
                name = mcpNameService.encode(NamesServiceImpl.PREFIX_WORKFLOW, "search", null);
            } else {
                String target = mcpNameService.encode(NamesServiceImpl.PREFIX_WORKFLOW, "search", null);
                Assert.assertEquals(name, target);
            }
        }
    }
}
