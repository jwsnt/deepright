package ai.deepright.complex.utils;

import org.apache.commons.lang3.StringUtils;

public class WhitespaceUtils {

    public static double score(String input) throws Exception {
        if (StringUtils.isBlank(input)) return 0.0;
        String[] lines = input.split("\\R");
        if (lines.length <= 1) return 0.0;
        int indentLines = 0;
        int listLines = 0;
        int nonEmptyLines = 0;
        for (String line : lines) {
            if (StringUtils.isBlank(line)) continue;
            nonEmptyLines++;
            if (line.startsWith("\t") || line.startsWith("  ")) {
                indentLines++;
            }
            String trimmed = line.trim();
            if (trimmed.matches("^([-*+]|\\d+\\.)\\s+.*")) {
                listLines++;
            }
        }
        double lineWeight = Math.min(nonEmptyLines / 5.0, 1.0) * 0.3;
        double indentWeight = (double) indentLines / nonEmptyLines * 0.4;
        double listWeight = (double) listLines / nonEmptyLines * 0.3;
        return lineWeight + indentWeight + listWeight;
    }
}