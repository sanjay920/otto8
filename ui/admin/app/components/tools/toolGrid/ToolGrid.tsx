import { useCallback, useEffect, useState } from "react";

import { ToolReference } from "~/lib/model/toolReferences";
import {
    ToolCategoryMap,
    YourToolsToolCategory,
} from "~/lib/service/api/toolreferenceService";

import { TypographyP } from "~/components/Typography";
import { CategoryHeader } from "~/components/tools/toolGrid/CategoryHeader";
import { CategoryTools } from "~/components/tools/toolGrid/CategoryTools";
import { useDebounce } from "~/hooks/useDebounce";

interface ToolGridProps {
    toolCategories: ToolCategoryMap;
    filter: string;
    onDelete: (id: string) => void;
}

export function ToolGrid({ toolCategories, filter, onDelete }: ToolGridProps) {
    const [filteredResults, setFilteredResults] =
        useState<ToolCategoryMap>(toolCategories);

    const filterCategories = useCallback(
        (searchTerm: string) => {
            const result: ToolCategoryMap = {};
            for (const [category, { tools, bundleTool }] of Object.entries(
                toolCategories
            )) {
                const sortedTools = tools.sort((a, b) =>
                    a.name.localeCompare(b.name)
                );
                const toolsWithBundle = bundleTool
                    ? [bundleTool, ...sortedTools]
                    : sortedTools;
                const filteredTools = toolsWithBundle.filter((tool) =>
                    [tool.name, tool.metadata?.category, tool.description]
                        .filter((x) => !!x)
                        .join("|")
                        .toLowerCase()
                        .includes(searchTerm.toLowerCase())
                );
                if (filteredTools.length > 0) {
                    result[category] = {
                        tools: filteredTools,
                        bundleTool: bundleTool,
                    };
                }
            }
            setFilteredResults(result);
        },
        [toolCategories]
    );

    const debouncedFilter = useDebounce(filterCategories, 150);

    useEffect(() => {
        debouncedFilter(filter);
    }, [filter, debouncedFilter]);

    if (!Object.entries(filteredResults).length) {
        return <TypographyP>No tools found...</TypographyP>;
    }

    const yourToolsCategory = filteredResults[YourToolsToolCategory];
    return (
        <div className="space-y-8 pb-16">
            {yourToolsCategory &&
                renderToolCategory(
                    YourToolsToolCategory,
                    yourToolsCategory.tools
                )}
            {Object.entries(filteredResults).map(
                ([category, { tools, bundleTool }]) => {
                    if (category === YourToolsToolCategory) return null;
                    return renderToolCategory(
                        category,
                        tools,
                        bundleTool?.description
                    );
                }
            )}
        </div>
    );

    function renderToolCategory(
        category: string,
        tools: ToolReference[],
        description = ""
    ) {
        if (!tools.length) return null;
        return (
            <div key={category} className="space-y-4">
                <CategoryHeader
                    category={category}
                    description={description}
                    tools={tools}
                />
                <CategoryTools tools={tools} onDelete={onDelete} />
            </div>
        );
    }
}
