package ai.open.right.workflow.flow.llm.provider.anthropic;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.Collections;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

@Setter
@Getter
public class AnthropicRequest extends ProviderRequest {

    public static final DefaultMedia EMPTY = new DefaultMedia();

    // { "type": "json_object" }
    protected Map<String, Object> responseFormat;

    protected Map<String, Object> cacheControl;

    protected Map<String, Object> thinking;

    protected AnthropicMedia anthropicMedia = AnthropicRequest.EMPTY;

    protected Double temperature;

    protected Integer maxTokens;

    protected String mimeType;

    protected String model;

    protected Double topP;

    @Override
    public Map<String, Object> getResponseSchema() {
        return this.responseFormat;
    }

    public static class DefaultMedia implements AnthropicMedia {

        protected static final Set<String> IMAGE_MIME = Collections.unmodifiableSet(Set.of(
                "image/jpeg",
                "image/bmp",
                "image/png"
        ));

        protected static final Set<String> PDF_TYPES = Collections.unmodifiableSet(Set.of(
                "application/pdf"
        ));


        protected final Set<String> mimeTypes = new HashSet<String>();

        protected static final String PREFIX = "inline:";

        public static final String IMAGE = "image";

        public DefaultMedia() {
            // 支持图片 PDF
            this.mimeTypes.addAll(DefaultMedia.IMAGE_MIME);
            this.mimeTypes.addAll(DefaultMedia.PDF_TYPES);
        }

        @Override
        public String getType(String type) throws Exception {
            this.checkValid(type);
            return StringUtils.contains(type, DefaultMedia.IMAGE) ? "image" : "document";
        }

        public Set<String> getMimes() throws Exception {
            return this.mimeTypes;
        }

        protected void checkValid(String type) throws Exception {
            // 如果是Base64参数为inline:xxx
            if (!this.mimeTypes.contains(this.trimType(type))) {
                throw new WorkflowException("The mime type is invalid: `" + type + "` and just support: " + ArrayUtils.toString(this.getMimes()), ProtocolCode.C915).needSilent();
            }
        }

        protected String trimType(String type) throws Exception {
            return StringUtils.startsWith(type, DefaultMedia.PREFIX) ? type.substring(PREFIX.length()) : type;
        }
    }
}
