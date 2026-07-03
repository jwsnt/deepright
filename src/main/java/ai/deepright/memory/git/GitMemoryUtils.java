package ai.deepright.memory.git;

import ai.deepright.lang.XmlResourceLang;
import ai.open.right.workflow.flow.WorkflowTask;
import org.apache.commons.lang3.StringUtils;

import java.util.ArrayList;
import java.util.List;

public class GitMemoryUtils {

    public static String buildMarkdown(WorkflowTask workTask, String content) throws Exception {
        StringBuffer title = new StringBuffer().append("|").append(XmlResourceLang.get(GitMemoryService.LANG_KEY_DATETIME)).append("|").append(XmlResourceLang.get(GitMemoryService.LANG_KEY_IMPORTANT)).append("|").append(XmlResourceLang.get(GitMemoryService.LANG_KEY_DIGEST)).append("|").append(XmlResourceLang.get(GitMemoryService.LANG_KEY_NEEDS)).append("|");
        StringBuilder buffer = new StringBuilder(title.toString()).append(System.lineSeparator()).append("|---|---|---|---|---|");
        for (String raw : StringUtils.defaultString(content).split("\\R")) {
            String line = StringUtils.trimToEmpty(raw);
            if (StringUtils.isBlank(line)) {
                continue;
            }
            List<String> sections = GitMemoryUtils.parseBracketSections(line);
            int offset = !sections.isEmpty() && StringUtils.isNumeric(sections.getFirst()) ? 1 : 0;
            if (sections.size() - offset < 3) {
                continue;
            }
            String datetime = GitMemoryUtils.readMemoryField(sections, offset, "datetime");
            String important = GitMemoryUtils.readMemoryField(sections, offset + 1, "important");
            String digest = GitMemoryUtils.readMemoryField(sections, offset + 2, null);
            String entity = GitMemoryUtils.readMemoryField(sections, offset + 3, null);
            String needs = GitMemoryUtils.readMemoryField(sections, offset + 4, null);
            buffer.append(System.lineSeparator()).append("|").append(GitMemoryUtils.escapeMarkdownCell(datetime)).append("|").append(GitMemoryUtils.escapeMarkdownCell(important)).append("|").append(GitMemoryUtils.escapeMarkdownCell(digest)).append("|").append(GitMemoryUtils.escapeMarkdownCell(entity)).append("|").append(GitMemoryUtils.escapeMarkdownCell(needs)).append("|");
        }
        return buffer.toString();
    }

    protected static List<String> parseBracketSections(String line) {
        List<String> sections = new ArrayList<String>();
        StringBuilder current = null;
        for (int i = 0; i < line.length(); i++) {
            char each = line.charAt(i);
            if (each == '[' && current == null) {
                current = new StringBuilder();
                continue;
            }
            if (each == ']' && current != null) {
                sections.add(current.toString());
                current = null;
                continue;
            }
            if (current != null) {
                current.append(each);
            }
        }
        return sections;
    }

    protected static String readMemoryField(List<String> sections, int index, String key) {
        if (index < 0 || index >= sections.size()) {
            return "";
        }
        String value = StringUtils.defaultString(sections.get(index));
        if (StringUtils.isBlank(key)) {
            return value;
        }
        String prefix = key + "=";
        return StringUtils.startsWith(value, prefix) ? StringUtils.substringAfter(value, prefix) : value;
    }

    protected static String escapeMarkdownCell(String value) {
        return StringUtils.defaultString(value).replace("|", "\\|").replace("\r", "").replace("\n", "<br>");
    }
}
