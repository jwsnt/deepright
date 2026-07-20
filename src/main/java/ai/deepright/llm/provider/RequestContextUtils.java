package ai.deepright.llm.provider;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import com.google.common.collect.ImmutableMap;
import org.apache.commons.lang3.StringUtils;

import java.util.LinkedHashMap;
import java.util.Map;

public class RequestContextUtils {

    protected static final Map<String, Integer> MODEL = new LinkedHashMap<String, Integer>();

    protected static final Double SHORT_BASE = 0.6d;

    protected static final Double LONG_BASE = 1.2d;

    static {
        // 有先后顺序
        RequestContextUtils.MODEL.put("mimo-v2-flash", 1024 * 256);
        RequestContextUtils.MODEL.put("mimo-v2-omni", 1024 * 256);
        RequestContextUtils.MODEL.put("deepseek", 1024 * 1024);
        RequestContextUtils.MODEL.put("minimax", 1024 * 200);
        RequestContextUtils.MODEL.put("gemini", 1024 * 1024);
        RequestContextUtils.MODEL.put("claude", 1024 * 1024);
        RequestContextUtils.MODEL.put("qwen3.5", 1024 * 256);
        RequestContextUtils.MODEL.put("qwen3.6", 1024 * 500);
        RequestContextUtils.MODEL.put("mimo", 1024 * 1024);
        RequestContextUtils.MODEL.put("kimi", 1024 * 256);
        RequestContextUtils.MODEL.put("gpt", 1024 * 256);
        RequestContextUtils.MODEL.put("glm", 1024 * 200);
    }

    public static void thinking(WorkflowTask workTask, String medium, String high) throws Exception {
        ComplexityMode complexity = ComplexityUtils.result(workTask);
        if (!ComplexityMode.FAST_REPLY.is(complexity)) {
            workTask.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_REASONING_EFFORT, ComplexityMode.TASK_PLANNING.is(complexity) ? medium : high);
            workTask.putMetadata(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_THINKING, ImmutableMap.of("type", "enabled"));
        }
    }

    public static Integer limit(WorkflowTask workTask, String model) throws Exception {
        Double base = ComplexityMode.FAST_REPLY.equals(ComplexityUtils.result(workTask)) ? RequestContextUtils.SHORT_BASE : RequestContextUtils.LONG_BASE;
        for (String key : RequestContextUtils.MODEL.keySet()) {
            if (StringUtils.containsIgnoreCase(model, key)) {
                return (int) (RequestContextUtils.MODEL.get(key) * base);
            }
        }
        return (int) (1024 * 200 * base);
    }
}
