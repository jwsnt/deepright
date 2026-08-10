package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.ObjectBuilder;
import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import org.junit.Assert;
import org.junit.Test;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;

/**
 * ProviderToken#selectToken 单测：空/单段/多段、trim、随机取其一。
 */
public class ProviderTokenTest {

    private static final ProviderRequest REQUEST = new OpenAiRequest();
    private static final LLMConfig LLM_CONFIG = null;
    private static final LLMQuery LLM_QUERY = ObjectBuilder.buildLLMQuery();
    private static final String API = "openai";

    private final ProviderToken providerToken = new ProviderToken();

    @Test
    public void selectToken_null_returnsNull() throws Exception {
        Assert.assertEquals("", providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, null));
    }

    @Test
    public void selectToken_emptyString_returnsEmpty() throws Exception {
        Assert.assertEquals("", providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, ""));
    }

    @Test
    public void selectToken_singleToken_noComma_returnsTrimmed() throws Exception {
        Assert.assertEquals("sk-one", providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, "sk-one"));
        Assert.assertEquals("sk-one", providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, "  sk-one  "));
    }

    @Test
    public void selectToken_twoTokens_returnsOneOfThem() throws Exception {
        Set<String> allowed = new HashSet<>(Arrays.asList("a", "b"));
        for (int i = 0; i < 20; i++) {
            String result = providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, "a,b");
            Assert.assertTrue("result should be one of a,b, got: " + result, allowed.contains(result));
        }
    }

    @Test
    public void selectToken_manyTokens_returnsOneOfThem() throws Exception {
        Set<String> allowed = new HashSet<>(Arrays.asList("x", "y", "z"));
        for (int i = 0; i < 30; i++) {
            String result = providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, "x,y,z");
            Assert.assertTrue("result should be one of x,y,z, got: " + result, allowed.contains(result));
        }
    }

    @Test
    public void selectToken_withSpacesAroundComma_returnsTrimmedPart() throws Exception {
        Set<String> allowed = new HashSet<>(Arrays.asList("key1", "key2"));
        for (int i = 0; i < 20; i++) {
            String result = providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, " key1 , key2 ");
            Assert.assertTrue("result should be trimmed key1 or key2, got: " + result, allowed.contains(result));
        }
    }

    @Test
    public void selectToken_singleToken_withSpaces_returnsTrimmed() throws Exception {
        Assert.assertEquals("sk-xxx", providerToken.select(REQUEST, LLM_CONFIG, LLM_QUERY, "  sk-xxx  "));
    }

    // ---------- InitConfig 单测 ----------

    @Test
    public void initConfig_providerToken_returnsNonNullBean() throws Exception {
        ProviderToken.InitConfig initConfig = new ProviderToken.InitConfig();
        ProviderToken bean = initConfig.providerToken();
        Assert.assertNotNull(bean);
        Assert.assertSame(ProviderToken.class, bean.getClass());
    }

    @Test
    public void initConfig_providerToken_beanSelectWorks() throws Exception {
        ProviderToken.InitConfig initConfig = new ProviderToken.InitConfig();
        ProviderToken bean = initConfig.providerToken();
        Assert.assertEquals("sk-one", bean.select(REQUEST, LLM_CONFIG, LLM_QUERY, "sk-one"));
    }
}
