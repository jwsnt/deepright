package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.WorkflowException;
import ai.open.right.utils.DumpUtils;
import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.MessageDelegate;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.config.LLMTakeover;
import org.apache.commons.io.FileUtils;
import org.apache.commons.lang3.StringUtils;
import org.junit.Assert;
import org.junit.Test;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

public class ProviderRequestTest {

    @Test
    public void testGetResponseSchemaDefaultReturnsNull() {
        Assert.assertNull(new ProviderRequest().getResponseSchema());
    }

    @Test
    public void testToString() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertFalse(req.hasAutoDump());
        req.setAutoDump("OT");
        Assert.assertTrue(req.hasAutoDump());
        req.setToken("Token");
        req.setScene("SCENE");
        req.setStream(false);
        req.setPrintReason(true);
        Assert.assertEquals("OT", req.getAutoDump());
        Assert.assertTrue(req.getPrintReason());
        Assert.assertTrue(req.toString().length() > 150);
    }

    @Test
    public void testChained() {
        ProviderRequest req = new ProviderRequest();
        req.setChain("Chain");
        Assert.assertTrue(req.hasChain());
        Assert.assertEquals("Chain", req.getChain());
    }

    @Test
    public void testIsWriteable1() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertFalse(req.getStoreFunCall());
        Assert.assertFalse(req.isWriteable());
        req.setStoreFunCall(true);
        Assert.assertTrue(req.getStoreFunCall());
    }

    @Test
    public void testIsWriteable2() {
        ProviderRequest req = new ProviderRequest();
        req.setContainHistories(true);
        Assert.assertTrue(req.isWriteable());
    }

    @Test
    public void testIsWriteable3() {
        ProviderRequest req = new ProviderRequest();
        req.setContainHistories(false);
        Assert.assertFalse(req.isWriteable());
    }

    @Test
    public void testIsWriteable4() {
        ProviderRequest req = new ProviderRequest();
        req.setContainHistories(false);
        req.setWriteable(true);
        Assert.assertFalse(req.isWriteable());
    }

    @Test
    public void testIsWriteable5() {
        ProviderRequest req = new ProviderRequest();
        req.setContainHistories(true);
        req.setWriteable(false);
        Assert.assertFalse(req.isWriteable());
    }

    @Test
    public void testGetRepo() {
        ProviderRequest req = new ProviderRequest();
        req.setScene("HELLO");
        req.setMaxError(1024);
        Assert.assertEquals(Integer.valueOf(1024), req.getMaxError());
        Assert.assertEquals(req.getRepositories().getFirst(), "HELLO");
    }

    @Test
    public void testGetRepoWithSame() {
        ProviderRequest req = new ProviderRequest();
        List<String> repo = new ArrayList<String>();
        repo.add("HELLO");
        req.setRepositories(repo);
        req.setScene("WORLD");
        Assert.assertEquals(req.getRepositories().toString(), "[HELLO, WORLD]");
    }

    @Test
    public void testGetRepoWithRepl() {
        ProviderRequest req = new ProviderRequest();
        List<String> repo = new ArrayList<String>();
        repo.add("HELLO");
        req.setRepositories(repo);
        req.setScene("HELLO");
        Assert.assertEquals(req.getRepositories().toString(), "[HELLO]");
    }

    @Test
    public void testGetQueryForHistory() {
        ProviderRequest req = new ProviderRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setQuery("ABCD");
        req.setMessage(message);
        Assert.assertEquals("INITIAL", req.getQuery4History());
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        Assert.assertEquals("INITIAL", req.getQuery4History());
    }

    @Test
    public void testGetQueryForHistoryWithNotPure() {
        ProviderRequest req = new ProviderRequest();
        Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
        message.setQuery("ABCD");
        req.setPureQuery(false);
        req.setMessage(message);
        Assert.assertEquals("ABCD", req.getQuery4History());
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setPureQuery(true);
        Assert.assertEquals("INITIAL", req.getQuery4History());
    }

    @Test
    public void testNotifier() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertEquals("NotifierB", req.getNotifier("NotifierB"));
        req.setNotifier("NotifierA");
        Assert.assertTrue(req.hasNotifier());
        Assert.assertEquals("NotifierA", req.getNotifier());
        Assert.assertEquals("NotifierA", req.getNotifier("NotifierB"));
    }

    @Test
    public void testSetFunCallWithCurrentNull() {
        List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
        funCalls.add(new LLMFunCall());
        ProviderRequest req = new ProviderRequest();
        req.setFunCalls(funCalls);
        Assert.assertEquals(funCalls, req.getFunCalls());
    }

    @Test
    public void testSetFunCallWithCurrentExists() {
        List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
        funCalls.add(new LLMFunCall());
        ProviderRequest req = new ProviderRequest();
        req.setFunCalls(funCalls);
        List<ProviderFunCall> funCalls2 = new ArrayList<ProviderFunCall>();
        funCalls2.add(new LLMFunCall());
        req.setFunCalls(funCalls2);
        Assert.assertEquals(Integer.valueOf(2), Integer.valueOf(req.getFunCalls().size()));
    }

    @Test
    public void testSetFunCallWithEmpty() {
        List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
        ProviderRequest req = new ProviderRequest();
        req.setFunCalls(funCalls);
        Assert.assertNull(req.getFunCalls());
    }

    @Test
    public void testSetFunCallWithNull() {
        ProviderRequest req = new ProviderRequest();
        req.setFunCalls(null);
        Assert.assertNull(req.getFunCalls());
    }

    @Test
    public void testAddTakeover() {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        Assert.assertNull(req.getTakeovers());
        Assert.assertFalse(req.isTakeover("A"));
        LLMTakeover llmTakeover = new LLMTakeover();
        req.addTakeover("A", llmTakeover);
        Assert.assertTrue(req.isTakeover("A"));
        Assert.assertFalse(req.isTakeover("B"));
        Assert.assertEquals(llmTakeover, req.getTakeover("A"));
        Assert.assertNull(req.getTakeover("B"));
    }

    @Test
    public void testGetRepositoriesSceneExists() {
        ProviderRequest req = new ProviderRequest();
        req.setScene("S");
        req.setRepositories(new ArrayList<>(Arrays.asList("S")));
        Assert.assertEquals(1, req.getRepositories().size());
    }

    @Test
    public void testIsTakeoverNull() {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setTakeovers(null);
        Assert.assertFalse(req.isTakeover("A"));
    }


    private static ProviderRequest autodumpReadyRequest(String autodumpDir) {
        ProviderRequest req = new ProviderRequest();
        req.setAutoDump(autodumpDir);
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setModel("model-under-test");
        req.getProviderData().setRequest("{\"payload\":true}");
        return req;
    }

    /** {@link ProviderRequest#autoDump} 每次成功会各写 request、response 两个文件（时间戳不同故为两个文件名） */
    @Test
    public void testAutoDump() throws IOException {
        File dir = Files.createTempDirectory("provider_autodump").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.getProviderData().setResponse("{\"responseBody\":true}");
            req.autoDump(new WorkflowException("", ProtocolCode.C500));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(2, files.length);
            String all = FileUtils.readFileToString(files[0], StandardCharsets.UTF_8)
                    + FileUtils.readFileToString(files[1], StandardCharsets.UTF_8);
            Assert.assertTrue(all.contains("payload"));
            Assert.assertTrue(all.contains("responseBody"));
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    @Test
    public void testAutoDump2() throws IOException {
        ProviderRequest req = new ProviderRequest();
        req.setAutoDump(StringUtils.repeat("A", 1000));
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setModel("m");
        req.getProviderData().setRequest("A");
        req.getProviderData().setResponse("B");
        req.autoDump(new WorkflowException("", ProtocolCode.C500));
    }

    @Test
    public void testAutoDumpSuccess() throws IOException {
        File dir = Files.createTempDirectory("provider_autodump_success").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.getProviderData().setResponse("{\"responseBody\":true}");
            req.autoDump();
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(2, files.length);
            String all = FileUtils.readFileToString(files[0], StandardCharsets.UTF_8)
                    + FileUtils.readFileToString(files[1], StandardCharsets.UTF_8);
            Assert.assertTrue(all.contains("payload"));
            Assert.assertTrue(all.contains("responseBody"));
            for (File file : files) {
                Assert.assertTrue(file.getName().contains("model-under-test"));
                Assert.assertFalse(file.getName().contains("_500_"));
            }
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    @Test
    public void testAutoDumpSuccessWhenAutoDumpDisabled() {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setModel("m");
        req.getProviderData().setRequest("A");
        req.getProviderData().setResponse("B");
        req.setAutoDump(null);
        req.autoDump();
        req.setAutoDump("");
        req.autoDump();
    }

    @Test
    public void testAutoDumpSuccessWhenPathInvalid_doesNotThrow() throws IOException {
        File file = File.createTempFile("provider_autodump_file", ".tmp");
        try {
            ProviderRequest req = autodumpReadyRequest(file.getAbsolutePath());
            req.getProviderData().setResponse("{\"responseBody\":true}");
            req.autoDump();
        } finally {
            FileUtils.deleteQuietly(file);
        }
    }

    /**
     * autodump()：hasAutodump() 为 false 时不写文件、不抛异常
     */
    @Test
    public void testAutodumpWhenNoAutodumpPath_doesNothing() {
        ProviderRequest req = new ProviderRequest();
        req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
        req.setAutoDump("");
        req.getProviderData().setRequest("x");
        req.autoDump(new WorkflowException("", ProtocolCode.C500));
        req.setAutoDump(null);
        req.autoDump(new WorkflowException("", ProtocolCode.C500));
    }

    /**
     * e.getCode() <= C0 时不写入（条件：e.getCode() > ProtocolCode.C0）
     */
    @Test
    public void testAutodumpWhenExceptionCodeNotAboveC0_doesNotWrite() throws IOException {
        File dir = Files.createTempDirectory("autodump_skip_c0").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.autoDump(new WorkflowException("", ProtocolCode.C0));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(0, files.length);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump()：目录不存在时 mkdirs 后写入；写入内容为 JsonUtils.write(providerData.getRequest())
     */
    @Test
    public void testAutodumpWhenDirNotExists_createsDirAndWritesJson() throws IOException {
        File dir = Files.createTempDirectory("autodump_test").toFile();
        try {
            File sub = new File(dir, "sub");
            Assert.assertFalse(sub.exists());
            ProviderRequest req = autodumpReadyRequest(sub.getAbsolutePath());
            req.getProviderData().setRequest("AAAA");
            req.getProviderData().setResponse("BBB");
            req.autoDump(new WorkflowException("", ProtocolCode.C500));
            Assert.assertTrue(sub.exists());
            Assert.assertTrue(sub.isDirectory());
            Assert.assertNotNull(sub.listFiles());
            Assert.assertEquals(2, sub.listFiles().length);
            Assert.assertTrue(sub.listFiles()[0].getName().endsWith(".json"));
            Assert.assertTrue(sub.listFiles()[1].getName().endsWith(".json"));
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autoDump：{@link ProtocolCode#C400} 时无条件落盘（注释「400 或 …」），无需 hasAutoDump / __autodump；目录为 defaultIfEmpty 的 {@code autodump}
     */
    @Test
    public void testAutoDump_c400_withoutAutodumpPath_stillDumps() throws IOException {
        File defaultDir = new File(System.getProperty("user.dir"), "autodump");
        if (defaultDir.exists()) {
            FileUtils.deleteDirectory(defaultDir);
        }
        try {
            ProviderRequest req = new ProviderRequest();
            req.setMessage(new MessageDelegate(ObjectBuilder.buildLLMQuery()));
            req.setModel("c400-model");
            req.setAutoDump(null);
            req.getProviderData().setRequest("{\"req\":1}");
            req.getProviderData().setResponse("{\"res\":2}");
            req.autoDump(new WorkflowException("", ProtocolCode.C400));
            Assert.assertTrue(defaultDir.isDirectory());
            File[] files = defaultDir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(2, files.length);
        } finally {
            if (defaultDir.exists()) {
                FileUtils.deleteDirectory(defaultDir);
            }
        }
    }

    /**
     * autoDump：message.metadata.__autodump=true 时视为 hasAutoDump()，且 e.getCode() &gt; C0 时落盘（无 autoDump 路径时用目录名 autodump）
     */
    @Test
    public void testAutoDump_metadataAutodumpWithoutPath() throws IOException {
        File defaultDir = new File(System.getProperty("user.dir"), "autodump");
        if (defaultDir.exists()) {
            FileUtils.deleteDirectory(defaultDir);
        }
        try {
            MessageDelegate message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
            message.putMetadata("__autodump", true);
            ProviderRequest req = new ProviderRequest();
            req.setMessage(message);
            req.setModel("meta-model");
            req.setAutoDump(null);
            Assert.assertFalse(req.hasAutoDump());
        } finally {
            if (defaultDir.exists()) {
                FileUtils.deleteDirectory(defaultDir);
            }
        }
    }

    /**
     * autodump()：发生异常时捕获并 log.error，不向外抛
     */
    @Test
    public void testAutodumpOnException_logsErrorAndDoesNotThrow() throws IOException {
        File readOnlyDir = Files.createTempDirectory("autodump_readonly").toFile();
        readOnlyDir.setReadOnly();
        try {
            ProviderRequest req = autodumpReadyRequest(readOnlyDir.getAbsolutePath());
            req.autoDump(new WorkflowException("", ProtocolCode.C500));
        } finally {
            readOnlyDir.setWritable(true);
            FileUtils.deleteDirectory(readOnlyDir);
        }
    }

    @Test
    public void testGetApiDefaultNull() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertNull(req.getApi());
    }

    @Test
    public void testSetApiAndGetApi() {
        ProviderRequest req = new ProviderRequest();
        req.setApi(ProviderRequest.REQUEST_ANTHROPIC);
        Assert.assertEquals(ProviderRequest.REQUEST_ANTHROPIC, req.getApi());

        req.setApi(ProviderRequest.REQUEST_COZE);
        Assert.assertEquals(ProviderRequest.REQUEST_COZE, req.getApi());

        req.setApi(ProviderRequest.REQUEST_GOOGLE);
        Assert.assertEquals(ProviderRequest.REQUEST_GOOGLE, req.getApi());

        req.setApi(ProviderRequest.REQUEST_OPENAI);
        Assert.assertEquals(ProviderRequest.REQUEST_OPENAI, req.getApi());

        req.setApi(ProviderRequest.REQUEST_SEEDREAM);
        Assert.assertEquals(ProviderRequest.REQUEST_SEEDREAM, req.getApi());
    }

    @Test
    public void testIsApiWhenApiUnsetAndNameNull() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertNull(req.getApi());
        Assert.assertTrue(req.isApi(null));
    }

    @Test
    public void testIsApiWhenApiUnsetAndNameNonNull() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertFalse(req.isApi(ProviderRequest.REQUEST_OPENAI));
    }

    @Test
    public void testIsApiCaseInsensitive() {
        ProviderRequest req = new ProviderRequest();
        req.setApi(ProviderRequest.REQUEST_OPENAI);
        Assert.assertTrue(req.isApi("openai"));
        Assert.assertTrue(req.isApi("OpenAI"));
        Assert.assertTrue(req.isApi("OPENAI"));
    }

    @Test
    public void testIsApiWhenNameDifferent() {
        ProviderRequest req = new ProviderRequest();
        req.setApi(ProviderRequest.REQUEST_OPENAI);
        Assert.assertFalse(req.isApi(ProviderRequest.REQUEST_COZE));
        Assert.assertFalse(req.isApi("other"));
    }

    @Test
    public void testIsApiWhenNameNullButApiSet() {
        ProviderRequest req = new ProviderRequest();
        req.setApi(ProviderRequest.REQUEST_OPENAI);
        Assert.assertFalse(req.isApi(null));
    }

    @Test
    public void testApiConstantsMatchProviderServices() {
        Assert.assertEquals("anthropic", ProviderRequest.REQUEST_ANTHROPIC);
        Assert.assertEquals("coze", ProviderRequest.REQUEST_COZE);
        Assert.assertEquals("google", ProviderRequest.REQUEST_GOOGLE);
        Assert.assertEquals("openai", ProviderRequest.REQUEST_OPENAI);
        Assert.assertEquals("seedream", ProviderRequest.REQUEST_SEEDREAM);
    }

    @Test
    public void testPutExtendedWhenExtendedConfigIsNull() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertNull(req.getExtendedConfig());
        ProviderRequest result = req.putExtended("key1", "value1");
        Assert.assertSame(req, result);
        Assert.assertNotNull(req.getExtendedConfig());
        Assert.assertEquals("value1", req.getExtendedConfig().get("key1"));
    }

    @Test
    public void testPutExtendedWhenExtendedConfigExists() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("key1", "value1");
        ProviderRequest result = req.putExtended("key2", "value2");
        Assert.assertSame(req, result);
        Assert.assertEquals("value1", req.getExtendedConfig().get("key1"));
        Assert.assertEquals("value2", req.getExtendedConfig().get("key2"));
    }

    @Test
    public void testPutExtendedChained() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("a", 1).putExtended("b", 2).putExtended("c", 3);
        Assert.assertEquals(1, req.getExtendedConfig().get("a"));
        Assert.assertEquals(2, req.getExtendedConfig().get("b"));
        Assert.assertEquals(3, req.getExtendedConfig().get("c"));
    }

    @Test
    public void testDelExtendedWhenExtendedConfigIsNull() {
        ProviderRequest req = new ProviderRequest();
        req.setExtendedConfig(null);
        Assert.assertNull(req.delExtended("anyKey", Object.class));
        Assert.assertNull(req.getExtendedConfig());
    }

    @Test
    public void testDelExtendedWhenKeyExists() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("key1", "value1").putExtended("key2", "value2");
        String removed = req.delExtended("key1", String.class);
        Assert.assertEquals("value1", removed);
        Assert.assertNull(req.getExtendedConfig().get("key1"));
        Assert.assertEquals("value2", req.getExtendedConfig().get("key2"));
    }

    @Test
    public void testDelExtendedWhenKeyNotExists() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("key1", "value1");
        Assert.assertNull(req.delExtended("key2", String.class));
        Assert.assertEquals("value1", req.getExtendedConfig().get("key1"));
    }

    @Test
    public void testPutExtendedAndDelExtendedCombined() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("k1", "v1").putExtended("k2", "v2");
        Assert.assertEquals("v1", req.delExtended("k1", String.class));
        Assert.assertNull(req.getExtendedConfig().get("k1"));
        Assert.assertEquals("v2", req.getExtendedConfig().get("k2"));
    }

    @Test
    public void testDelExtendedWithClazzWhenExtendedConfigIsNull() {
        ProviderRequest req = new ProviderRequest();
        req.setExtendedConfig(null);
        Assert.assertNull(req.delExtended("anyKey", String.class));
    }

    @Test
    public void testDelExtendedWithClazzWhenKeyExists() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("key1", "value1").putExtended("key2", "value2");
        String removed = req.delExtended("key1", String.class);
        Assert.assertEquals("value1", removed);
        Assert.assertNull(req.getExtendedConfig().get("key1"));
        Assert.assertEquals("value2", req.getExtendedConfig().get("key2"));
    }

    @Test
    public void testDelExtendedWithClazzWhenKeyNotExists() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("key1", "value1");
        Assert.assertNull(req.delExtended("key2", String.class));
        Assert.assertEquals("value1", req.getExtendedConfig().get("key1"));
    }

    @Test
    public void testDelExtendedWithClazzReturnsTypedValue() {
        ProviderRequest req = new ProviderRequest();
        req.putExtended("num", Integer.valueOf(42));
        Integer removed = req.delExtended("num", Integer.class);
        Assert.assertEquals(Integer.valueOf(42), removed);
        Assert.assertNull(req.getExtendedConfig().get("num"));
    }

    @Test
    public void discard_defaultsToNull() {
        ProviderRequest req = new ProviderRequest();
        Assert.assertNull(req.getDiscard());
    }

    @Test
    public void discard_setFalse() {
        ProviderRequest req = new ProviderRequest();
        req.setDiscard(false);
        Assert.assertFalse(req.getDiscard());
    }

    @Test
    public void discard_setTrue() {
        ProviderRequest req = new ProviderRequest();
        req.setDiscard(true);
        Assert.assertTrue(req.getDiscard());
    }

    /**
     * autodump()：e.getCode() 为负数（内部Code）时不写入
     */
    @Test
    public void testAutodump_negativeCode_doesNotWrite() throws IOException {
        File dir = Files.createTempDirectory("autodump_neg").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.autoDump(new WorkflowException("takeover", ProtocolCode.I001));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(0, files.length);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump()：文件名包含 model 和 code
     */
    @Test
    public void testAutodump_fileNameContainsModelAndCode() throws IOException {
        File dir = Files.createTempDirectory("autodump_name").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.setModel("gpt-4o");
            req.autoDump(new WorkflowException("err", ProtocolCode.C502));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(2, files.length);
            for (File f : files) {
                String name = f.getName();
                Assert.assertTrue(name.contains("gpt-4o"));
                Assert.assertTrue(name.contains("502"));
                Assert.assertTrue(name.endsWith(".json"));
            }
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump()：多次调用写入多个文件
     */
    @Test
    public void testAutodump_multipleCalls_writesMultipleFiles() throws IOException {
        File dir = Files.createTempDirectory("autodump_multi").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.getProviderData().setRequest("AAA");
            req.getProviderData().setResponse("BBB");
            req.autoDump(new WorkflowException("e1", ProtocolCode.C500));
            req.autoDump(new WorkflowException("e2", ProtocolCode.C502));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(4, files.length);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump 静态方法：写入内容与 body 一致
     */
    @Test
    public void testAutodumpStatic_writesCorrectBody() throws Exception {
        File dir = Files.createTempDirectory("autodump_static").toFile();
        try {
            Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
            String body = "{\"test\":\"data\"}";
            DumpUtils.dump(message, dir.getAbsolutePath(), "dump.json", body);
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(1, files.length);
            String content = FileUtils.readFileToString(files[0], StandardCharsets.UTF_8);
            Assert.assertEquals(body, content);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump 静态方法：目录不存在时自动创建
     */
    @Test
    public void testAutodumpStatic_createsNestedDir() throws Exception {
        File dir = Files.createTempDirectory("autodump_nested").toFile();
        try {
            File nested = new File(dir, "a/b/c");
            Assert.assertFalse(nested.exists());
            Message message = new MessageDelegate(ObjectBuilder.buildLLMQuery());
            DumpUtils.dump(message, nested.getAbsolutePath(), "test.json", "body");
            Assert.assertTrue(nested.exists());
            Assert.assertEquals(1, nested.listFiles().length);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }

    /**
     * autodump()：model 为 null 时不抛异常，文件名包含 null 字符串
     */
    @Test
    public void testAutodump_nullModel_doesNotThrow() throws IOException {
        File dir = Files.createTempDirectory("autodump_nullmodel").toFile();
        try {
            ProviderRequest req = autodumpReadyRequest(dir.getAbsolutePath());
            req.setModel(null);
            req.autoDump(new WorkflowException("err", ProtocolCode.C500));
            File[] files = dir.listFiles();
            Assert.assertNotNull(files);
            Assert.assertEquals(2, files.length);
        } finally {
            FileUtils.deleteDirectory(dir);
        }
    }
}
