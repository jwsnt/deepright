package ai.deepright.llm.provider;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.complex.ComplexityMode;
import ai.deepright.complex.ComplexityUtils;
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

    static {
        RequestModelSelect.OUTPUT.add("media@image_gen");
        RequestModelSelect.INPUT.add("media@file_gen");
        RequestModelSelect.INPUT.add("media@ocr_gen");
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

    public static boolean multiOutput(WorkflowTask workTask) throws Exception {
        return RequestModelSelect.OUTPUT.contains(SplitUtils.join(workTask));
    }

    public static boolean multiInput(WorkflowTask workTask) throws Exception {
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
