package ai.deepright.memory;

import ai.deepright.lang.XmlResourceLang;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.util.Assert;

import java.util.List;

@Getter
@Setter
public class MemoryRecall {

    protected static final char[] INVALID_KEYWORD_CHARS = new char[]{'\\', '.', '^', '$', '|', '?', '*', '+', '(', ')', '[', ']', '{', '}', '\''};

    protected List<String> keywords;

    protected String before;

    protected String after;

    public Boolean hasKeyword() {
        return !CollectionUtils.isEmpty(this.keywords) && this.keywords.stream().anyMatch(this::isValidKeyword);
    }

    public Boolean hasBefore() {
        return !StringUtils.isEmpty(this.before);
    }

    public Boolean hasAfter() {
        return !StringUtils.isEmpty(this.after);
    }

    public String buildKeywords() throws Exception {
        StringBuilder buffer = new StringBuilder();
        if (!CollectionUtils.isEmpty(this.keywords)) {
            for (String each : this.keywords) {
                if (this.isValidKeyword(each)) {
                    buffer.append(each).append("|");
                }
            }
        }
        return buffer.isEmpty() ? buffer.toString() : buffer.substring(0, buffer.length() - 1);
    }

    public void checkValid() throws Exception {
        Assert.isTrue(this.hasKeyword() || this.hasBefore() || this.hasAfter(), XmlResourceLang.get(MemoryService.LANG_KEY_MEMORY_RECALL_VALID));
    }

    protected boolean isValidKeyword(String keyword) {
        return !StringUtils.isBlank(keyword) && !StringUtils.containsAny(keyword, MemoryRecall.INVALID_KEYWORD_CHARS);
    }
}
