package ai.open.right.workflow.flow.llm.provider.google;

import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;
import org.springframework.util.StringUtils;

import java.net.URI;


public class VertexTokenTest {

    @Test
    public void testCreate() throws Exception {
        String file = System.getenv("PROVIDER_VERTEX_TOKEN_URI");
        if (StringUtils.hasText(file)) {
            String token = VertexToken.token(IOUtils.toByteArray(URI.create(file).toURL().openStream()), null, 100);
            Assert.assertNotNull(token);
            Assert.assertFalse(token.isEmpty());
        }
    }
}
