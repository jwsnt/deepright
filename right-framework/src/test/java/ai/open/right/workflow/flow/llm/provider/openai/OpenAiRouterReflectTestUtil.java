package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.workflow.flow.llm.LLMCallback;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import org.apache.http.client.methods.HttpPost;

import java.lang.reflect.Method;

/** 测试用反射调用 {@link OpenAiRouter} 的 protected reader / reConfig。 */
public final class OpenAiRouterReflectTestUtil {

    private static final Method READER;
    private static final Method RECONFIG;

    static {
        try {
            READER = OpenAiRouter.class.getDeclaredMethod(
                    "reader", OpenAiRequest.class, LLMConfig.class, LLMCallback.class);
            READER.setAccessible(true);
            RECONFIG = OpenAiRouter.class.getDeclaredMethod(
                    "reConfig", OpenAiRequest.class, LLMConfig.class, HttpPost.class);
            RECONFIG.setAccessible(true);
        } catch (ReflectiveOperationException e) {
            throw new ExceptionInInitializerError(e);
        }
    }

    private OpenAiRouterReflectTestUtil() {
    }

    public static OpenAiReader invokeReader(
            OpenAiRouter router,
            OpenAiRequest request,
            LLMConfig config,
            LLMCallback callback) throws Exception {
        return (OpenAiReader) READER.invoke(router, request, config, callback);
    }

    public static void invokeReConfig(
            OpenAiRouter router,
            OpenAiRequest request,
            LLMConfig config,
            HttpPost post) throws Exception {
        RECONFIG.invoke(router, request, config, post);
    }
}
