package ai.open.right.workflow.flow.llm.provider.deepseek;

import ai.open.right.workflow.flow.llm.LLMFunCallRequest;
import ai.open.right.workflow.flow.llm.LLMFunCallResponse;
import ai.open.right.workflow.flow.llm.provider.openai.OpenAiRequest;
import ai.open.right.workflow.flow.llm.store.history.History;

import java.lang.reflect.Constructor;
import java.lang.reflect.Method;
import java.util.List;

/**
 * DeepSeek 已继承 {@link ai.open.right.workflow.flow.llm.provider.openai.OpenAiRouter}，原 DeepSeekMessage / DeepSeekContent
 * 与 OpenAi 共用内嵌类；测试通过反射访问父类内嵌类，避免直接引用已删除的符号。
 */
public final class DeepSeekRouterReflectTestUtil {

    private static final Class<?> OPEN_AI_ROUTER = DeepSeekRouter.class.getSuperclass();

    private static final Class<?> OPEN_AI_MESSAGE;
    private static final Class<?> OPEN_AI_CONTENT;

    static {
        OPEN_AI_MESSAGE = requireNested("OpenAiMessage");
        OPEN_AI_CONTENT = requireNested("OpenAiContent");
    }

    private DeepSeekRouterReflectTestUtil() {
    }

    private static Class<?> requireNested(String simpleName) {
        for (Class<?> c : OPEN_AI_ROUTER.getDeclaredClasses()) {
            if (simpleName.equals(c.getSimpleName())) {
                return c;
            }
        }
        throw new IllegalStateException("Nested class not found: " + simpleName + " on " + OPEN_AI_ROUTER.getName());
    }

    public static Object newOpenAiMessage(OpenAiRequest openAiRequest) throws Exception {
        Constructor<?> ctor = OPEN_AI_MESSAGE.getDeclaredConstructor(OpenAiRequest.class);
        ctor.setAccessible(true);
        return ctor.newInstance(openAiRequest);
    }

    public static Object newOpenAiContent(LLMFunCallRequest request) throws Exception {
        Constructor<?> ctor = OPEN_AI_CONTENT.getDeclaredConstructor(LLMFunCallRequest.class);
        ctor.setAccessible(true);
        return ctor.newInstance(request);
    }

    public static Object newOpenAiContent(LLMFunCallResponse response) throws Exception {
        Constructor<?> ctor = OPEN_AI_CONTENT.getDeclaredConstructor(LLMFunCallResponse.class);
        ctor.setAccessible(true);
        return ctor.newInstance(response);
    }

    public static Object newOpenAiContent(History history) throws Exception {
        Constructor<?> ctor = OPEN_AI_CONTENT.getDeclaredConstructor(History.class);
        ctor.setAccessible(true);
        return ctor.newInstance(history);
    }

    public static Object getReasoning(Object openAiContent) throws Exception {
        return OPEN_AI_CONTENT.getMethod("getReasoning").invoke(openAiContent);
    }

    public static Object getThinking(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getThinking").invoke(openAiMessage);
    }

    public static Object getResponseFormat(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getResponseFormat").invoke(openAiMessage);
    }

    public static Object getFrequencyPenalty(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getFrequencyPenalty").invoke(openAiMessage);
    }

    public static Object getTemperature(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getTemperature").invoke(openAiMessage);
    }

    public static Object getStream(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getStream").invoke(openAiMessage);
    }

    public static Object getModel(Object openAiMessage) throws Exception {
        return OPEN_AI_MESSAGE.getMethod("getModel").invoke(openAiMessage);
    }

    @SuppressWarnings("unchecked")
    public static List<Object> getMessages(Object openAiMessage) throws Exception {
        return (List<Object>) OPEN_AI_MESSAGE.getMethod("getMessages").invoke(openAiMessage);
    }

    public static Object getContent(Object openAiContent) throws Exception {
        return OPEN_AI_CONTENT.getMethod("getContent").invoke(openAiContent);
    }

    public static Object getRole(Object openAiContent) throws Exception {
        return OPEN_AI_CONTENT.getMethod("getRole").invoke(openAiContent);
    }

    public static void invokeBuildFunCallData(Object openAiMessage, OpenAiRequest request) throws Exception {
        Method m = OPEN_AI_MESSAGE.getDeclaredMethod("buildFunCallData", OpenAiRequest.class);
        m.setAccessible(true);
        m.invoke(openAiMessage, request);
    }
}
