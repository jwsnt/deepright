package ai.open.right.workflow.config.impl;

import ai.open.right.WorkflowException;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class NamesServiceImplTest {

    @Test
    public void testEncodeDecode() throws Exception {
        NamesServiceImpl service = new NamesServiceImpl();
        service.init();
        service.setEncode(true);
        service.setLength(8);
        
        String prefix = NamesServiceImpl.PREFIX_TOOLS;
        String client = "client";
        String name = "name";
        
        String encoded = service.encode(prefix, client, name);
        Assertions.assertTrue(encoded.startsWith(prefix));
        
        String[] decoded = service.decode(encoded);
        Assertions.assertEquals(2, decoded.length);
        Assertions.assertEquals(client, decoded[0]);
        Assertions.assertEquals(name, decoded[1]);
    }

    @Test
    public void testEncodeNoChange() throws Exception {
        NamesServiceImpl service = new NamesServiceImpl();
        service.setEncode(false);
        String input = "test";
        Assertions.assertEquals(input, service.encode(input));
    }

    @Test
    public void testIsPrefix() throws Exception {
        NamesServiceImpl service = new NamesServiceImpl();
        service.init();
        Assertions.assertTrue(service.isPrefix(NamesServiceImpl.PREFIX_TOOLS + "test"));
        Assertions.assertTrue(service.isPrefix(NamesServiceImpl.PREFIX_WORKFLOW + "test"));
        Assertions.assertTrue(service.isPrefix(NamesServiceImpl.PREFIX_PROMPT + "test"));
        Assertions.assertTrue(service.isPrefix(NamesServiceImpl.PREFIX_RESOURCE + "test"));
        Assertions.assertFalse(service.isPrefix("test"));
    }

    @Test
    public void testDecodeError() throws Exception {
        NamesServiceImpl service = new NamesServiceImpl();
        service.init();
        // 强制使用Tools解析，并解析错误
        Assertions.assertThrows(WorkflowException.class, () -> {
            service.decode("invalid");
        });
    }

    @Test
    public void testInitConfig() throws Exception {
        NamesServiceImpl.InitConfig config = new NamesServiceImpl.InitConfig();
        config.setEncode(true);
        config.setLength(16);
        Assertions.assertNotNull(config.namesService());
    }
}
