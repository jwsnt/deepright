package ai.open.right.workflow.flow.llm.provider.openai;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.*;

@Setter
@Getter
public class OpenAiRequest extends ProviderRequest {

    // { "type": "json_object" }
    protected Map<String, Object> responseFormat;

    protected OpenAiMedia openAiMedia = DefaultMedia.DEFAULT;

    protected Double frequencyPenalty;

    protected Double presencePenalty;

    protected String reasoningEffort;

    protected Boolean funCallStream;

    protected Double temperature;

    protected Integer maxTokens;

    protected String mimeType;

    protected Double topP;

    @Override
    public Map<String, Object> getResponseSchema() {
        return this.responseFormat;
    }

    // https://developers.openai.com/api/docs/guides/file-inputs
    // 文件解析
    public static class DefaultMedia implements OpenAiMedia {

        protected static final Set<String> IMAGE_MIME = Collections.unmodifiableSet(Set.of(
                "image/jpeg",
                "image/bmp",
                "image/png"
        ));

        protected final Set<String> mimeTypes = new HashSet<String>();

        protected static final String PREFIX = "inline:";

        public static final DefaultMedia DEFAULT = new DefaultMedia();

        public DefaultMedia() {
            this.mimeTypes.addAll(DefaultMedia.IMAGE_MIME);
        }

        @Override
        public String getPrefix(String type) throws Exception {
            return "data:" + type + ";base64,";
        }

        @Override
        public String getKeyUrl(String type) throws Exception {
            // 仅支持图片
            this.checkValid(type);
            return "image_url";
        }

        @Override
        public Set<String> getMimes() throws Exception {
            return this.mimeTypes;
        }

        protected void checkValid(String type) throws Exception {
            // 如果是Base64参数为inline:xxx
            if (!this.mimeTypes.contains(this.trimType(type))) {
                throw new WorkflowException("The mime type is invalid: `" + type + "` and just support: " + ArrayUtils.toString(this.getMimes())).needSilent();
            }
        }

        protected String trimType(String type) throws Exception {
            return StringUtils.startsWith(type, DefaultMedia.PREFIX) ? type.substring(PREFIX.length()) : type;
        }
    }
}
