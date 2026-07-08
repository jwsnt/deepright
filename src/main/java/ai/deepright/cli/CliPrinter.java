package ai.deepright.cli;

import static org.springframework.util.ObjectUtils.isEmpty;

import ai.deepright.llm.notifier.MultiSourceFlag;
import org.apache.commons.lang3.StringUtils;

import java.util.HashMap;
import java.util.Map;

public class CliPrinter {

    public static final Integer BRIEF = 100;

    public static final Integer SIZE_L = 4;

    public static final Integer SIZE_S = 6;

    public static final Integer SIZE_N = 0;

    public static Boolean includeChineseMoreThanEnglish(String content) {
        for (int i = 0; i < content.length(); i++) {
            char ch = content.charAt(i);
            Character.UnicodeBlock ub = Character.UnicodeBlock.of(ch);
            if (ub == Character.UnicodeBlock.CJK_UNIFIED_IDEOGRAPHS
                    || ub == Character.UnicodeBlock.CJK_COMPATIBILITY_IDEOGRAPHS
                    || ub == Character.UnicodeBlock.CJK_UNIFIED_IDEOGRAPHS_EXTENSION_A) {
                return true;
            }
        }
        return false;
    }

    public static Boolean isLastCharPunctuation(String content) throws Exception {
        // 获取最后一个字符
        char lastChar = content.charAt(content.length() - 1);
        // 获取字符的 Unicode 类型
        int type = Character.getType(lastChar);
        // 判断是否属于 Unicode 标点符号的分类
        // 连接符，如 '_'
        return type == Character.CONNECTOR_PUNCTUATION ||
                // 破折号/连字符，如 '-'
                type == Character.DASH_PUNCTUATION ||
                // 前括号，如 '(', '[', '《'
                type == Character.START_PUNCTUATION ||
                // 后括号，如 ')', ']', '》'
                type == Character.END_PUNCTUATION ||
                // 前引号，如 '“'
                type == Character.INITIAL_QUOTE_PUNCTUATION ||
                // 后引号，如 '”'
                type == Character.FINAL_QUOTE_PUNCTUATION ||
                // 其他标点，如 '!', '?', '。', '，'
                type == Character.OTHER_PUNCTUATION;
    }

    public static String format(String content, Integer repeat) throws Exception {
        if (!StringUtils.isEmpty(content)) {
            StringBuffer buffer = new StringBuffer(System.lineSeparator());
            buffer.append(StringUtils.repeat("#", repeat)).append(!CliPrinter.SIZE_N.equals(repeat) ? " " : "").append(content);
            buffer.append(!StringUtils.endsWith(content, System.lineSeparator()) ? System.lineSeparator() : "");
            String result = buffer.toString();
            // 检查时使用trim
            String trim = StringUtils.trim(result);
            return CliPrinter.isLastCharPunctuation(trim) ? result : trim + (CliPrinter.includeChineseMoreThanEnglish(result) ? "。" : ".");
        } else {
            return "";
        }
    }

    public static Map<String, Object> process(String scene, String key, String val) throws Exception {
        Map<String, Object> result = new HashMap<String, Object>();
        result.put(MultiSourceFlag.PROCESS, scene);
        result.put(key, val);
        return result;
    }

    public static Map<String, Object> process(String scene, String agent) throws Exception {
        Map<String, Object> result = new HashMap<String, Object>();
        result.put(MultiSourceFlag.PROCESS, scene);
        result.put(MultiSourceFlag.TARGET, agent);
        return result;
    }

    public static Map<String, Object> process(String scene) throws Exception {
        Map<String, Object> result = new HashMap<String, Object>();
        result.put(MultiSourceFlag.PROCESS, scene);
        return result;
    }

    public static String image(String content) throws Exception {
        // ![](http://127.0.0.1:9998/data?name=xxxx)
        StringBuffer buffer = new StringBuffer();
        buffer.append("![](").append(content).append(")").append(System.lineSeparator());
        return buffer.toString();
    }

    public static String brief(String content) throws Exception {
        StringBuffer buffer = new StringBuffer();
        // maxWidth >= marker.length() + 1
        buffer.append(System.lineSeparator()).append(StringUtils.abbreviate(content, "...", Math.max(CliPrinter.BRIEF, 4)));
        buffer.append(!StringUtils.endsWith(content, System.lineSeparator()) ? System.lineSeparator() : "");
        return buffer.toString();
    }

    public static String code(String content) throws Exception {
        StringBuffer buffer = new StringBuffer(System.lineSeparator());
        buffer.append(System.lineSeparator());
        buffer.append("```").append(System.lineSeparator());
        buffer.append(content).append(!StringUtils.endsWith(content, System.lineSeparator()) ? System.lineSeparator() : "");
        buffer.append("```").append(System.lineSeparator());
        buffer.append(System.lineSeparator());
        return buffer.toString();
    }
}
