package ai.deepright.llm.provider;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.feature.FeatureUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.LLMQueryService;
import org.apache.commons.lang3.StringUtils;

import java.util.LinkedHashMap;
import java.util.Map;

public class RequestProviderUtils {

    protected static Map<String, String> PROVIDER = new LinkedHashMap<String, String>();

    static {
        RequestProviderUtils.PROVIDER.put("kimi", LLMQueryService.LLM_KIMI);
        RequestProviderUtils.PROVIDER.put("mimo", LLMQueryService.LLM_KIMI);
        RequestProviderUtils.PROVIDER.put("qwen", LLMQueryService.LLM_QWEN);
        RequestProviderUtils.PROVIDER.put("openai", LLMQueryService.LLM_OPENAI);
        RequestProviderUtils.PROVIDER.put("vertex", LLMQueryService.LLM_VERTEX);
        RequestProviderUtils.PROVIDER.put("gemini", LLMQueryService.LLM_GEMINI);
        RequestProviderUtils.PROVIDER.put("minimax", LLMQueryService.LLM_MINIMAX);
        RequestProviderUtils.PROVIDER.put("deepseek", LLMQueryService.LLM_DEEPSEEK);
        RequestProviderUtils.PROVIDER.put("bigmodel", LLMQueryService.LLM_BIGMODEL);
        RequestProviderUtils.PROVIDER.put("anthropic", LLMQueryService.LLM_ANTHROPIC);
        RequestProviderUtils.PROVIDER.put("volcengine", LLMQueryService.LLM_VOLCENGINE);
    }

    // 是否为多模态模型
    public static Boolean isMultiInputModel(WorkflowTask workTask) throws Exception {
        String provider = RequestProviderUtils.findProvider(workTask);
        // 为空，或者为Gemini或Vertex/Kimi/BigModel
        return StringUtils.isEmpty(provider) ||
                (
                        // A：图片 PDf
                        // G/V/B：全模态
                        // O：图片
                        // X：图 视频 音频
                        // K：图 视频 音频
                        // M：图 视频
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_VOLCENGINE) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_ANTHROPIC) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_BIGMODEL) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_MINIMAX) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_GEMINI) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_VERTEX) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_OPENAI) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_XIAOMI) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_QWEN) ||
                        StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_KIMI)
                );
    }

    // 是否为多模态模型
    public static Boolean isMultiOutputModel(WorkflowTask workTask) throws Exception {
        return RequestProviderUtils.isMultiOutputModel(RequestProviderUtils.findProvider(workTask));
    }

    public static Boolean isMultiOutputModel(String provider) throws Exception {
        // 为空，或者为Gemini或Vertex
        return StringUtils.isEmpty(provider) ||
                (
                    StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_GEMINI) ||
                    StringUtils.equalsIgnoreCase(provider, LLMQueryService.LLM_VERTEX)
                );
    }

    public static String findProvider(WorkflowTask workTask) throws Exception {
        String provider = FeatureUtils.buildTargetProvider(workTask);
        for (String each : RequestProviderUtils.PROVIDER.keySet()) {
            if (StringUtils.startsWith(provider, each)) {
                return RequestProviderUtils.PROVIDER.get(each);
            }
        }
        return provider;
    }
}
