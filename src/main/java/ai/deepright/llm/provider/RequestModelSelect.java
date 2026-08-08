package ai.deepright.llm.provider;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
import ai.open.right.WorkflowException;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import lombok.Builder;
import lombok.Getter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.HashSet;
import java.util.Map;
import java.util.Set;

public class RequestModelSelect {

    protected static final Set<String> OUTPUT = new HashSet<String>();

    protected static final Set<String> INPUT = new HashSet<String>();

    public static final String KEY_MODEL_MULTI_OUTPUT = "__model_multi_output";

    public static final String KEY_MODEL_MULTI_INPUT = "__model_multi_input";

    public static final String KEY_MODEL_THINKING = "__model_thinking";

    public static final String KEY_MODEL_FAST = "__model_fast";

    public static final String KEY_MODEL_BASE = "__model";

    public static final String KEY_MODEL_URL = "__url";

    public static final String KEY_PROXY = "@";

    static {
        RequestModelSelect.OUTPUT.add("media@image_gen");
        RequestModelSelect.INPUT.add("media@file_gen");
        RequestModelSelect.INPUT.add("media@ocr_gen");
    }

    public static Boolean isProxyMultiOutput(Map<String, Object> metadata) throws Exception {
        return StringUtils.startsWith(MapUtils.getString(metadata, RequestModelSelect.KEY_MODEL_MULTI_OUTPUT), RequestModelSelect.KEY_PROXY);
    }

    public static Boolean isProxyMultiOutput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.isProxyMultiOutput(workTask.getMetadata());
    }

    public static Boolean isProxyMultiInput(Map<String, Object> metadata) throws Exception {
        return StringUtils.startsWith(MapUtils.getString(metadata, RequestModelSelect.KEY_MODEL_MULTI_INPUT), RequestModelSelect.KEY_PROXY);
    }

    public static Boolean isProxyMultiInput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.isProxyMultiInput(workTask.getMetadata());
    }

    public static Boolean isProxyAvailable(WorkflowTask workTask) throws Exception {
        return (RequestModelSelect.multiOutput(workTask) && RequestModelSelect.isProxyMultiOutput(workTask)) || (RequestModelSelect.multiInput(workTask) && RequestModelSelect.isProxyMultiInput(workTask));
    }

    public static String proxyMultiOutput(Map<String, Object> metadata) throws Exception {
        String provider = MapUtils.getString(metadata, RequestModelSelect.KEY_MODEL_MULTI_OUTPUT);
        // 代理模型，不为空值时就必须@开头
        WorkflowException.checkCondition(!StringUtils.isEmpty(provider) && !StringUtils.startsWith(provider, RequestModelSelect.KEY_PROXY), "The proxy multi output model is invalid");
        return StringUtils.substring(provider, RequestModelSelect.KEY_PROXY.length());
    }

    public static String proxyMultiOutput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.proxyMultiOutput(workTask.getMetadata());
    }

    public static String proxyMultiInput(Map<String, Object> metadata) throws Exception {
        String provider = MapUtils.getString(metadata, RequestModelSelect.KEY_MODEL_MULTI_INPUT);
        // 代理模型，不为空值时就必须@开头
        WorkflowException.checkCondition(!StringUtils.isEmpty(provider) && !StringUtils.startsWith(provider, RequestModelSelect.KEY_PROXY), "The proxy multi input model is invalid");
        return StringUtils.substring(provider, RequestModelSelect.KEY_PROXY.length());
    }

    public static String proxyMultiInput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.proxyMultiInput(workTask.getMetadata());
    }

    public static Map<String, Object> transfer(WorkflowTask workTask, Map<String, Object> metadata) throws Exception {
        String token = MapUtils.getString(workTask.getMetadata(), ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN);
        String multiOutput = MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_MULTI_OUTPUT);
        String multiInput = MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_MULTI_INPUT);
        String thinking = MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_THINKING);
        String provider = MapUtils.getString(workTask.getMetadata(), ProviderRequestService.KEY_PROVIDER);
        String base = MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_BASE);
        String fast = MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_FAST);
        String url = MapUtils.getString(workTask.getMetadata(), "__url");
        if (!StringUtils.isEmpty(multiOutput)) {
            metadata.put(RequestModelSelect.KEY_MODEL_MULTI_OUTPUT, multiOutput);
        }
        if (!StringUtils.isEmpty(multiInput)) {
            metadata.put(RequestModelSelect.KEY_MODEL_MULTI_INPUT, multiInput);
        }
        if (!StringUtils.isEmpty(thinking)) {
            metadata.put(RequestModelSelect.KEY_MODEL_THINKING, thinking);
        }
        if (!StringUtils.isEmpty(provider)) {
            metadata.put(ProviderRequestService.KEY_PROVIDER, provider);
        }
        if (!StringUtils.isEmpty(base)) {
            metadata.put(RequestModelSelect.KEY_MODEL_BASE, base);
        }
        if (!StringUtils.isEmpty(fast)) {
            metadata.put(RequestModelSelect.KEY_MODEL_FAST, fast);
        }
        if (!StringUtils.isEmpty(token)) {
            metadata.put(ProviderRequestService.KEY_INTERNAL + ProviderRequestService.KEY_TOKEN, token);
        }
        if (!StringUtils.isEmpty(url)) {
            metadata.put(RequestModelSelect.KEY_MODEL_URL, url);
        }
        return metadata;
    }

    public static String select(WorkflowTask workTask, RequestModel model) throws Exception {
        if (RequestModelSelect.multiOutput(workTask)) {
            return model.getMultiOutput(workTask);
        }
        if (RequestModelSelect.multiInput(workTask)) {
            return model.getMultiInput(workTask);
        }
        // 获取当前复杂度
        ComplexityMode lastMode = ComplexityUtils.result(workTask);
        if (lastMode.is(ComplexityMode.TASK_PLANNING, ComplexityMode.DEEP_THINKING)) {
            return model.getThinking(workTask);
        }
        if (lastMode.is(ComplexityMode.FAST_REPLY)) {
            return model.getFast(workTask);
        }
        return model.getBase(workTask);
    }

    public static Boolean multiOutput(String workflow) throws Exception {
        return RequestModelSelect.OUTPUT.contains(workflow);
    }

    public static Boolean multiOutput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.OUTPUT.contains(SplitUtils.join(workTask));
    }

    public static Boolean multiInput(String workflow) throws Exception {
        return RequestModelSelect.INPUT.contains(workflow);
    }

    public static Boolean multiInput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.INPUT.contains(SplitUtils.join(workTask));
    }

    @Builder
    @Getter
    public static class RequestModel {

        protected String multiOutput;

        protected String multiInput;

        protected String thinking;

        protected String fast;

        protected String base;

        public String getMultiOutput(WorkflowTask workTask) throws Exception {
            return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_MULTI_OUTPUT), this.multiOutput);
        }

        public String getMultiInput(WorkflowTask workTask) throws Exception {
            return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_MULTI_INPUT), this.multiInput);
        }

        public String getThinking(WorkflowTask workTask) throws Exception {
            return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_THINKING), this.thinking);
        }

        public String getFast(WorkflowTask workTask) throws Exception {
            return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_FAST), this.fast);
        }

        public String getBase(WorkflowTask workTask) throws Exception {
            return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), RequestModelSelect.KEY_MODEL_BASE), this.base);
        }
    }
}
