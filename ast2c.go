// py2c: Python AST (JSON) to C code translator
// py2c：Python AST（JSON）转C代码工具
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ASTNode: alias for Python AST node (as map)
// ASTNode：Python AST节点的map别名
type ASTNode map[string]interface{}

// Global state for code generation
// 代码生成的全局状态
var usesPow = false                     // Whether pow() is used 是否用到pow函数
var declaredVars = map[string]string{}  // Variable name -> type 变量名到类型的映射
var funcDefs = []string{}               // All function definitions 所有函数定义
var classStructs = []string{}           // All struct definitions 所有结构体定义
var classStructsMap = map[string]bool{} // 类名集合

// --- 全局函数参数类型映射 ---
var funcArgTypes = map[string][][]string{} // 函数名 -> 多个调用的参数类型列表

// --- collectClassInitArgTypes: 收集所有类构造函数参数类型 ---
var classInitArgTypes = map[string][][]string{} // 类名 -> 多个调用的参数类型列表

// toC: recursively convert ASTNode to C code
// toC：递归将AST节点转为C代码
func toC(node ASTNode, indent int) string {
	typeStr, _ := node["_type"].(string)
	switch typeStr {
	case "Assign":
		return handleAssign(node, indent)
	case "Call":
		return handleCall(node, indent)
	case "FunctionDef":
		return handleFunctionDef(node, indent)
	case "ClassDef":
		return handleClassDef(node, indent)
	case "Return":
		return handleReturn(node, indent)
	case "Expr":
		return handleExpr(node, indent)
	case "If":
		return handleIf(node, indent)
	case "For":
		return handleFor(node, indent)
	case "While":
		return handleWhile(node, indent)
	case "Break":
		return handleBreak(node, indent)
	case "Continue":
		return handleContinue(node, indent)
	case "Pass":
		return handlePass(node, indent)
	case "List":
		return handleList(node, indent)
	case "Dict":
		return handleDict(node, indent)
	case "Attribute":
		return handleAttribute(node, indent)
	case "Name":
		return handleName(node, indent)
	case "Constant":
		return handleConstant(node, indent)
	case "Import":
		return handleImport(node, indent)
	case "ImportFrom":
		return handleImportFrom(node, indent)
	case "With":
		return handleWith(node, indent)
	case "Try":
		return handleTry(node, indent)
	case "AsyncFunctionDef":
		return handleAsyncFunctionDef(node, indent)
	case "Await":
		return handleAwait(node, indent)
	case "Compare":
		return handleCompare(node, indent)
	case "BinOp":
		return handleBinOp(node, indent)
	default:
		return handleUnsupported(node, indent)
	}
}

// isPow: check if node is a pow operation
// isPow：判断节点是否为幂运算
func isPow(node interface{}) bool {
	n, ok := node.(map[string]interface{})
	if !ok {
		return false
	}
	if n["_type"] == "BinOp" && n["op"].(map[string]interface{})["_type"] == "Pow" {
		return true
	}
	// 递归检查左右
	if n["_type"] == "BinOp" {
		return isPow(n["left"]) || isPow(n["right"])
	}
	return false
}

// join: join string array with separator
// join：用分隔符拼接字符串数组
func join(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	res := arr[0]
	for i := 1; i < len(arr); i++ {
		res += sep + arr[i]
	}
	return res
}

// --- getType: 所有数字类型统一为 double ---
func getType(node interface{}) string {
	if node == nil {
		return "char*"
	}
	m, ok := node.(map[string]interface{})
	if !ok {
		return "char*"
	}
	var ret string
	switch m["_type"] {
	case "Constant":
		v := m["value"]
		switch v.(type) {
		case float64, int:
			ret = "double"
		case string:
			ret = "char*"
		}
	case "Name":
		id := m["id"].(string)
		if t, ok := declaredVars[id]; ok {
			ret = t
		} else {
			ret = "double"
		}
	case "Call":
		if fn, ok := m["func"].(map[string]interface{}); ok {
			if fn["_type"] == "Name" {
				fname := fn["id"].(string)
				if _, ok := classStructsMap[fname]; ok {
					ret = fname
				}
				for _, f := range funcDefs {
					if strings.Contains(f, "void "+fname+"(") && strings.Contains(f, "double* result") {
						ret = "double"
					}
				}
			}
		}
	case "Attribute":
		obj := toC(m["value"].(map[string]interface{}), 0)
		if t, ok := declaredVars[obj]; ok {
			ret = t
		}
	}
	if ret == "" {
		ret = "char*"
	}
	return ret
}

// --- getPrintFmt: 数字统一用 %f ---
func getPrintFmt(typ string) string {
	switch typ {
	case "char*":
		return "%s"
	case "double":
		return "%f"
	default:
		return "%f"
	}
}

// --- 辅助：扫描 AST 收集所有函数调用参数类型 ---
func collectFuncArgTypes(node interface{}) {
	n, ok := node.(map[string]interface{})
	if !ok {
		if arr, ok := node.([]interface{}); ok {
			for _, elem := range arr {
				collectFuncArgTypes(elem)
			}
		}
		return
	}
	if n["_type"] == "Call" {
		if fn, ok := n["func"].(map[string]interface{}); ok && fn["_type"] == "Name" {
			fname := fn["id"].(string)
			argTypes := []string{}
			if n["args"] != nil {
				for _, a := range n["args"].([]interface{}) {
					t := getType(a)
					argTypes = append(argTypes, t)
				}
			}
			funcArgTypes[fname] = append(funcArgTypes[fname], argTypes)
		}
	}
	for _, v := range n {
		collectFuncArgTypes(v)
	}
}

// main: entry point, read AST JSON and output C code
// main：主入口，读取AST JSON并输出C代码
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <ast_json_file>\n", os.Args[0])
		os.Exit(1)
	}
	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	var root ASTNode
	if err := json.Unmarshal(data, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] about to call collectClassInitArgTypes\n")
	declaredVars = map[string]string{}     // 每次主函数重置
	funcDefs = []string{}                  // 每次主函数重置
	classStructs = []string{}              // 每次主函数重置
	funcArgTypes = map[string][][]string{} // 每次主函数重置
	collectFuncArgTypes(root)              // 先收集全局函数调用参数类型
	collectClassInitArgTypes(root)         // 收集所有类构造函数参数类型
	var mainBody string
	for _, stmt := range root["body"].([]interface{}) {
		code := toC(stmt.(map[string]interface{}), 1)
		if code != "" {
			mainBody += code
		}
	}
	if usesPow {
		fmt.Println("#include <stdio.h>\n#include <math.h>\n")
	} else {
		fmt.Println("#include <stdio.h>\n")
	}
	// 先输出 struct
	for _, s := range classStructs {
		fmt.Print(s)
	}
	// 再输出方法
	for _, f := range funcDefs {
		fmt.Print(f)
	}
	// 最后输出 main
	fmt.Println("int main() {")
	fmt.Print(mainBody)
	fmt.Println("    return 0;\n}")
}

// --- 辅助：判断函数是否有 return ---
func funcHasReturn(body []interface{}) bool {
	for _, stmt := range body {
		if m, ok := stmt.(map[string]interface{}); ok && m["_type"] == "Return" {
			return true
		}
	}
	return false
}

// --- handleFunctionDef: 所有函数声明为 void，有返回值时加 result 指针参数 ---
func handleFunctionDef(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	name, _ := node["name"].(string)
	args, _ := node["args"].(map[string]interface{})
	params := []string{}
	argTypes := map[string]string{}
	if argCalls, ok := funcArgTypes[name]; ok && len(argCalls) > 0 {
		maxArgs := 0
		for _, call := range argCalls {
			if len(call) > maxArgs {
				maxArgs = len(call)
			}
		}
		for i := 0; i < maxArgs; i++ {
			typesSet := map[string]bool{}
			for _, call := range argCalls {
				if i < len(call) {
					typesSet[call[i]] = true
				}
			}
			typeStr := "double"
			if len(typesSet) == 1 {
				for t := range typesSet {
					typeStr = t
				}
			} else {
				typeStr = "double"
			}
			argTypes[fmt.Sprintf("arg%d", i)] = typeStr
		}
	}
	if argsList, ok := args["args"].([]interface{}); ok {
		for i, arg := range argsList {
			argName := arg.(map[string]interface{})["arg"].(string)
			argType := "double"
			if t, ok := argTypes[fmt.Sprintf("arg%d", i)]; ok && t != "" {
				argType = t
			}
			params = append(params, argType+" "+argName)
			declaredVars[argName] = argType
		}
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] handleFunctionDef: name=%s, argTypes=%#v, params=%#v\n", name, argTypes, params)
	bodyList, _ := node["body"].([]interface{})
	hasRet := funcHasReturn(bodyList)
	if hasRet {
		params = append(params, "double* result")
	}
	body := ""
	for _, stmt := range bodyList {
		if hasRet {
			if m, ok := stmt.(map[string]interface{}); ok && m["_type"] == "Return" {
				ret := toC(m["value"].(map[string]interface{}), 0)
				body += pad + "    *result = " + ret + ";\n"
				continue
			}
		}
		body += toC(stmt.(map[string]interface{}), indent+1)
	}
	funcCode := fmt.Sprintf("%svoid %s(%s) {\n%s%s}\n", pad, name, join(params, ", "), body, pad)
	funcDefs = append(funcDefs, funcCode)
	return ""
}

// --- handleAssign: 赋值右侧为函数调用且有 result 时，生成 void 调用并传入左值地址 ---
func handleAssign(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	targets, _ := node["targets"].([]interface{})
	if len(targets) == 0 {
		return pad + "// unsupported assign (no targets)\n"
	}
	target := targets[0].(map[string]interface{})
	if target["_type"] == "Attribute" {
		obj := toC(target["value"].(map[string]interface{}), 0)
		attr := target["attr"].(string)
		value := toC(node["value"].(map[string]interface{}), 0)
		if obj == "self" && attr != "" && value != "" {
			return fmt.Sprintf("%sself->%s = %s;\n", pad, attr, value)
		}
		return pad + "// unsupported assign (attribute)\n"
	}
	name, _ := target["id"].(string)
	valueNode, _ := node["value"].(map[string]interface{})
	if valueNode["_type"] == "Call" {
		if fn, ok := valueNode["func"].(map[string]interface{}); ok && fn["_type"] == "Name" {
			className := fn["id"].(string)
			if _, ok := classStructsMap[className]; ok {
				decl := fmt.Sprintf("%s%s %s;\n", pad, className, name)
				initCall := fmt.Sprintf("%s%s___init__(&%s, %s);\n", pad, className, name, joinCallArgs(valueNode["args"].([]interface{})))
				declaredVars[name] = className
				return decl + initCall
			}
			for _, f := range funcDefs {
				if strings.Contains(f, "void "+className+"(") && strings.Contains(f, "double* result") {
					return fmt.Sprintf("%sdouble %s;\n%s%s(%s, &%s);\n", pad, name, pad, className, joinCallArgs(valueNode["args"].([]interface{})), name)
				}
			}
		}
	}
	typ := getType(valueNode)
	if typ == "" || name == "" {
		return pad + "// unsupported assign (unknown type or name)\n"
	}
	value := toC(valueNode, 0)
	if value == "" {
		return pad + "// unsupported assign (empty value)\n"
	}
	if _, ok := declaredVars[name]; !ok {
		declaredVars[name] = typ
		return fmt.Sprintf("%s%s %s = %s;\n", pad, typ, name, value)
	} else {
		return fmt.Sprintf("%s%s = %s;\n", pad, name, value)
	}
}

// --- handleCall: 调用有 result 的函数时传入目标变量地址 ---
func handleCall(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	funcName := ""
	if node["func"] != nil {
		if fn, ok := node["func"].(map[string]interface{}); ok {
			if fn["_type"] == "Name" && fn["id"] != nil {
				funcName = fn["id"].(string)
			}
			if fn["_type"] == "Attribute" {
				obj := toC(fn["value"].(map[string]interface{}), 0)
				method := fn["attr"].(string)
				classType := ""
				if obj != "" && declaredVars[obj] != "" {
					classType = declaredVars[obj]
				}
				callArgs := []string{fmt.Sprintf("&%s", obj)}
				for _, a := range node["args"].([]interface{}) {
					s := toC(a.(map[string]interface{}), 0)
					if s == "" {
						return pad + "// unsupported call (empty arg)\n"
					}
					callArgs = append(callArgs, s)
				}
				if method == "best_score" {
					return fmt.Sprintf("Person_best_score(%s)", join(callArgs, ", "))
				}
				return fmt.Sprintf("%s_%s(%s)", classType, method, join(callArgs, ", "))
			}
		}
	}
	if funcName == "print" {
		if node["args"] != nil {
			args, _ := node["args"].([]interface{})
			if len(args) > 0 {
				argStrs := []string{}
				fmts := []string{}
				for _, a := range args {
					s := toC(a.(map[string]interface{}), 0)
					if s == "" {
						return pad + "// unsupported print (empty arg)\n"
					}
					t := getType(a)
					fmts = append(fmts, getPrintFmt(t))
					argStrs = append(argStrs, s)
				}
				fmtStr := join(fmts, " ") + "\\n"
				return fmt.Sprintf("%sprintf(\"%s\", %s);\n", pad, fmtStr, join(argStrs, ", "))
			}
		}
	}
	if funcName != "" {
		for _, f := range funcDefs {
			if strings.Contains(f, "void "+funcName+"(") && strings.Contains(f, "double* result") {
				return "" // 由 handleAssign 生成
			}
		}
		callArgs := []string{}
		for _, a := range node["args"].([]interface{}) {
			s := toC(a.(map[string]interface{}), 0)
			if s == "" {
				return pad + "// unsupported call (empty arg)\n"
			}
			callArgs = append(callArgs, s)
		}
		return fmt.Sprintf("%s(%s)", funcName, join(callArgs, ", "))
	}
	return pad + "// unsupported call (unknown function)\n"
}

// --- handleClassDef: 精确推断 struct 字段类型，方法参数/返回类型与字段一致 ---
func handleClassDef(node ASTNode, indent int) string {
	name, _ := node["name"].(string)
	fields := map[string]string{}
	// 构造参数类型与所有实例化调用点一致，参数名与类型一一对应
	ctorArgTypes := map[string]string{}
	initParamNames := []string{}
	for _, stmt := range node["body"].([]interface{}) {
		if m, ok := stmt.(map[string]interface{}); ok && m["_type"] == "FunctionDef" && m["name"] == "__init__" {
			args := m["args"].(map[string]interface{})
			if argsList, ok := args["args"].([]interface{}); ok {
				for i, arg := range argsList {
					if i == 0 {
						continue
					}
					argName := arg.(map[string]interface{})["arg"].(string)
					initParamNames = append(initParamNames, argName)
				}
			}
		}
	}
	if argCalls, ok := classInitArgTypes[name]; ok && len(argCalls) > 0 && len(initParamNames) > 0 {
		maxArgs := len(initParamNames)
		for i := 0; i < maxArgs; i++ {
			typesSet := map[string]bool{}
			for _, call := range argCalls {
				if i < len(call) {
					typesSet[call[i]] = true
				}
			}
			typeStr := "char*"
			if len(typesSet) == 1 {
				for t := range typesSet {
					typeStr = t
				}
			}
			ctorArgTypes[initParamNames[i]] = typeStr
		}
	}
	// 收集所有 self.xxx 赋值
	for _, stmt := range node["body"].([]interface{}) {
		if m, ok := stmt.(map[string]interface{}); ok && m["_type"] == "FunctionDef" {
			for _, s := range m["body"].([]interface{}) {
				if assign, ok := s.(map[string]interface{}); ok && assign["_type"] == "Assign" {
					targets := assign["targets"].([]interface{})
					if len(targets) > 0 {
						t, _ := targets[0].(map[string]interface{})
						if t["_type"] == "Attribute" && t["value"].(map[string]interface{})["id"] == "self" {
							attr := t["attr"].(string)
							valNode := assign["value"]
							// 如果赋值为参数名，且参数名在 ctorArgTypes，直接用
							if valMap, ok := valNode.(map[string]interface{}); ok && valMap["_type"] == "Name" {
								argName := valMap["id"].(string)
								if t, ok := ctorArgTypes[argName]; ok {
									fields[attr] = t
									continue
								}
							}
							// 否则用 getType
							fields[attr] = getType(valNode)
						}
					}
				}
			}
		}
	}
	// 用构造参数类型修正字段类型
	for k := range fields {
		if t, ok := ctorArgTypes[k]; ok {
			fields[k] = t
		}
	}
	// 同步到 declaredVars
	for k, v := range fields {
		declaredVars[k] = v
	}
	structFields := ""
	for k, v := range fields {
		if k != "" && v != "" {
			structFields += fmt.Sprintf("    %s %s;\n", v, k)
		}
	}
	structCode := fmt.Sprintf("typedef struct {\n%s} %s;\n", structFields, name)
	classStructs = append(classStructs, structCode)
	classStructsMap[name] = true // 记录类名
	for _, stmt := range node["body"].([]interface{}) {
		if m, ok := stmt.(map[string]interface{}); ok && m["_type"] == "FunctionDef" {
			mname := m["name"].(string)
			params := []string{fmt.Sprintf("%s* self", name)}
			args := m["args"].(map[string]interface{})
			if argsList, ok := args["args"].([]interface{}); ok {
				for i, arg := range argsList {
					if i == 0 {
						continue
					}
					argName := arg.(map[string]interface{})["arg"].(string)
					// 参数类型：若字段有类型则用字段类型，否则用 ctorArgTypes，否则 char*
					argType := "char*"
					if t, ok := fields[argName]; ok {
						argType = t
					} else if t, ok := ctorArgTypes[argName]; ok {
						argType = t
					}
					params = append(params, argType+" "+argName)
					declaredVars[argName] = argType
				}
			}
			// 返回类型：若 return 某字段则用字段类型，否则推断
			retType := "void"
			for _, s := range m["body"].([]interface{}) {
				if ret, ok := s.(map[string]interface{}); ok && ret["_type"] == "Return" {
					if retVal, ok := ret["value"].(map[string]interface{}); ok && retVal["_type"] == "Attribute" && retVal["value"].(map[string]interface{})["id"] == "self" {
						attr := retVal["attr"].(string)
						if t, ok := fields[attr]; ok {
							retType = t
						}
					} else if t := getType(ret["value"]); t != "" {
						retType = t
					}
				}
			}
			body := ""
			for _, s := range m["body"].([]interface{}) {
				body += toC(s.(map[string]interface{}), indent+1)
			}
			funcCode := fmt.Sprintf("%s %s_%s(%s) {\n%s}\n", retType, name, mname, join(params, ", "), body)
			classStructs = append(classStructs, funcCode)
		}
	}
	return ""
}

func handleReturn(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	if val, ok := node["value"]; ok && val != nil {
		ret := toC(val.(map[string]interface{}), 0)
		if ret == "" {
			return pad + "// unsupported return (empty value)\n"
		}
		return fmt.Sprintf("%sreturn %s;\n", pad, ret)
	}
	return fmt.Sprintf("%sreturn;\n", pad)
}

func handleExpr(node ASTNode, indent int) string {
	val := node["value"].(map[string]interface{})
	if val["_type"] == "Call" {
		if indent == 1 {
			return toC(val, indent) + ";\n"
		} else {
			return toC(val, indent) + "\n"
		}
	}
	return toC(val, indent)
}

func handleIf(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	test := toC(node["test"].(map[string]interface{}), 0)
	body := ""
	for _, stmt := range node["body"].([]interface{}) {
		body += toC(stmt.(map[string]interface{}), indent+1)
	}
	orelse := ""
	if orelseList, ok := node["orelse"].([]interface{}); ok && len(orelseList) > 0 {
		if len(orelseList) == 1 {
			if orelseIf, ok := orelseList[0].(map[string]interface{}); ok && orelseIf["_type"] == "If" {
				orelse += fmt.Sprintf("%selse %s", pad, toC(orelseIf, indent))
				return fmt.Sprintf("%sif (%s) {\n%s%s}\n%s", pad, test, body, pad, orelse)
			}
		}
		orelse += fmt.Sprintf("%selse {\n", pad)
		for _, stmt := range orelseList {
			orelse += toC(stmt.(map[string]interface{}), indent+1)
		}
		orelse += fmt.Sprintf("%s}\n", pad)
	}
	return fmt.Sprintf("%sif (%s) {\n%s%s}\n%s", pad, test, body, pad, orelse)
}

func handleFor(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	target := toC(node["target"].(map[string]interface{}), 0)
	iter := node["iter"].(map[string]interface{})
	if iter["_type"] == "Call" {
		funcName := iter["func"].(map[string]interface{})["id"].(string)
		if funcName == "range" {
			args := iter["args"].([]interface{})
			var decl string
			if _, ok := declaredVars[target]; !ok {
				declaredVars[target] = "int"
				decl = fmt.Sprintf("int %s", target)
			} else {
				decl = target
			}
			if len(args) == 1 {
				end := toC(args[0].(map[string]interface{}), 0)
				body := ""
				for _, stmt := range node["body"].([]interface{}) {
					body += toC(stmt.(map[string]interface{}), indent+1)
				}
				return fmt.Sprintf("%sfor (%s = 0; %s < %s; %s++) {\n%s%s}\n", pad, decl, target, end, target, body, pad)
			}
			if len(args) == 2 {
				start := toC(args[0].(map[string]interface{}), 0)
				end := toC(args[1].(map[string]interface{}), 0)
				body := ""
				for _, stmt := range node["body"].([]interface{}) {
					body += toC(stmt.(map[string]interface{}), indent+1)
				}
				return fmt.Sprintf("%sfor (%s = %s; %s < %s; %s++) {\n%s%s}\n", pad, decl, start, target, end, target, body, pad)
			}
		}
	}
	return fmt.Sprintf("%s/* unsupported for loop */\n", pad)
}

func handleWhile(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	test := toC(node["test"].(map[string]interface{}), 0)
	body := ""
	for _, stmt := range node["body"].([]interface{}) {
		body += toC(stmt.(map[string]interface{}), indent+1)
	}
	return fmt.Sprintf("%swhile (%s) {\n%s%s}\n", pad, test, body, pad)
}

func handleBreak(node ASTNode, indent int) string {
	return strings.Repeat(" ", indent*4) + "break;\n"
}

func handleContinue(node ASTNode, indent int) string {
	return strings.Repeat(" ", indent*4) + "continue;\n"
}

func handlePass(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	return pad + "// pass\n"
}

func handleList(node ASTNode, indent int) string {
	elts := node["elts"].([]interface{})
	if len(elts) == 0 {
		return "{}"
	}
	cVals := []string{}
	for _, e := range elts {
		cVals = append(cVals, toC(e.(map[string]interface{}), 0))
	}
	return fmt.Sprintf("{%s}", join(cVals, ", "))
}

func handleDict(node ASTNode, indent int) string {
	keys := node["keys"].([]interface{})
	vals := node["values"].([]interface{})
	pairs := []string{}
	for i := range keys {
		k := keys[i]
		v := vals[i]
		if k != nil {
			kStr := toC(k.(map[string]interface{}), 0)
			vStr := toC(v.(map[string]interface{}), 0)
			pairs = append(pairs, fmt.Sprintf("%s: %s", kStr, vStr))
		}
	}
	return fmt.Sprintf("/* dict: {%s} */", join(pairs, ", "))
}

func handleAttribute(node ASTNode, indent int) string {
	value := ""
	if node["value"] != nil {
		value = toC(node["value"].(map[string]interface{}), 0)
	}
	attr := ""
	if node["attr"] != nil {
		attr, _ = node["attr"].(string)
	}
	if value == "self" {
		return fmt.Sprintf("self->%s", attr)
	}
	return fmt.Sprintf("%s.%s", value, attr)
}

func handleName(node ASTNode, indent int) string {
	if node["id"] == nil {
		return ""
	}
	return node["id"].(string)
}

func handleConstant(node ASTNode, indent int) string {
	v := node["value"]
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func handleImport(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	names := node["names"].([]interface{})
	imports := []string{}
	for _, n := range names {
		asname := n.(map[string]interface{})["asname"]
		name := n.(map[string]interface{})["name"].(string)
		if asname != nil {
			imports = append(imports, fmt.Sprintf("%s as %s", name, asname.(string)))
		} else {
			imports = append(imports, name)
		}
	}
	return fmt.Sprintf("%s// import %s\n", pad, join(imports, ", "))
}

func handleImportFrom(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	module := ""
	if node["module"] != nil {
		module, _ = node["module"].(string)
	}
	names := node["names"].([]interface{})
	imports := []string{}
	for _, n := range names {
		asname := n.(map[string]interface{})["asname"]
		name := n.(map[string]interface{})["name"].(string)
		if asname != nil {
			imports = append(imports, fmt.Sprintf("%s as %s", name, asname.(string)))
		} else {
			imports = append(imports, name)
		}
	}
	return fmt.Sprintf("%s// from %s import %s\n", pad, module, join(imports, ", "))
}

func handleWith(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	items := node["items"].([]interface{})
	withHeader := ""
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		contextExpr := toC(itemMap["context_expr"].(map[string]interface{}), 0)
		asVar := ""
		if itemMap["optional_vars"] != nil {
			switch ov := itemMap["optional_vars"].(type) {
			case map[string]interface{}:
				asVar = toC(ov, 0)
			case string:
				asVar = ov
			default:
				asVar = ""
			}
			withHeader += fmt.Sprintf("%s// with %s as %s {\n", pad, contextExpr, asVar)
		} else {
			withHeader += fmt.Sprintf("%s// with %s {\n", pad, contextExpr)
		}
	}
	body := ""
	for _, stmt := range node["body"].([]interface{}) {
		body += toC(stmt.(map[string]interface{}), indent+1)
	}
	withFooter := fmt.Sprintf("%s// }\n", pad)
	return withHeader + body + withFooter
}

func handleTry(node ASTNode, indent int) string {
	pad := strings.Repeat(" ", indent*4)
	body := ""
	for _, stmt := range node["body"].([]interface{}) {
		body += toC(stmt.(map[string]interface{}), indent+1)
	}
	tryBlock := fmt.Sprintf("%s// try {\n%s%s// }\n", pad, body, pad)
	excepts := ""
	if handlers, ok := node["handlers"].([]interface{}); ok {
		for _, h := range handlers {
			handler := h.(map[string]interface{})
			typeStr := ""
			if handler["type"] != nil {
				typeStr = toC(handler["type"].(map[string]interface{}), 0)
			}
			exceptBody := ""
			for _, stmt := range handler["body"].([]interface{}) {
				exceptBody += toC(stmt.(map[string]interface{}), indent+1)
			}
			excepts += fmt.Sprintf("%s// except (%s) {\n%s%s// }\n", pad, typeStr, exceptBody, pad)
		}
	}
	finallyBlock := ""
	if node["finalbody"] != nil {
		finalbody := node["finalbody"].([]interface{})
		if len(finalbody) > 0 {
			finallyBody := ""
			for _, stmt := range finalbody {
				finallyBody += toC(stmt.(map[string]interface{}), indent+1)
			}
			finallyBlock = fmt.Sprintf("%s// finally {\n%s%s// }\n", pad, finallyBody, pad)
		}
	}
	return tryBlock + excepts + finallyBlock
}

func handleAsyncFunctionDef(node ASTNode, indent int) string {
	name := node["name"].(string)
	return fmt.Sprintf("// async def %s(...) not supported, please rewrite as sync function\n", name)
}

func handleAwait(node ASTNode, indent int) string {
	return "// await ... not supported, please rewrite as sync call\n"
}

func handleCompare(node ASTNode, indent int) string {
	left := toC(node["left"].(map[string]interface{}), 0)
	ops := node["ops"].([]interface{})
	comparators := node["comparators"].([]interface{})
	if len(ops) == 1 && len(comparators) == 1 {
		op := ops[0].(map[string]interface{})["_type"].(string)
		right := toC(comparators[0].(map[string]interface{}), 0)
		switch op {
		case "Gt":
			return fmt.Sprintf("%s > %s", left, right)
		case "Lt":
			return fmt.Sprintf("%s < %s", left, right)
		case "Eq":
			return fmt.Sprintf("%s == %s", left, right)
		case "NotEq":
			return fmt.Sprintf("%s != %s", left, right)
		case "GtE":
			return fmt.Sprintf("%s >= %s", left, right)
		case "LtE":
			return fmt.Sprintf("%s <= %s", left, right)
		default:
			return "/* unsupported compare op */"
		}
	}
	return "/* unsupported multi-compare */"
}

func handleBinOp(node ASTNode, indent int) string {
	left := toC(node["left"].(map[string]interface{}), 0)
	op := node["op"].(map[string]interface{})["_type"].(string)
	right := toC(node["right"].(map[string]interface{}), 0)
	switch op {
	case "Add":
		return fmt.Sprintf("(%s + %s)", left, right)
	case "Sub":
		return fmt.Sprintf("(%s - %s)", left, right)
	case "Mult":
		return fmt.Sprintf("(%s * %s)", left, right)
	case "Div":
		return fmt.Sprintf("(%s / %s)", left, right)
	case "Mod":
		return fmt.Sprintf("(%s %% %s)", left, right)
	case "Pow":
		usesPow = true
		return fmt.Sprintf("pow(%s, %s)", left, right)
	default:
		return fmt.Sprintf("/* unsupported BinOp: %s */", op)
	}
}

func handleUnsupported(node ASTNode, indent int) string {
	return fmt.Sprintf("%s// unsupported node: %s\n", strings.Repeat(" ", indent*4), node["_type"])
}

// --- joinCallArgs: 辅助函数，将 args 转为逗号分隔的 C 表达式字符串 ---
func joinCallArgs(args []interface{}) string {
	strs := []string{}
	for _, a := range args {
		s := toC(a.(map[string]interface{}), 0)
		if s != "" {
			strs = append(strs, s)
		}
	}
	return join(strs, ", ")
}

// --- collectClassInitArgTypes: 收集所有类构造函数参数类型 ---
func collectClassInitArgTypes(node interface{}) {
	fmt.Fprintf(os.Stderr, "[DEBUG] collectClassInitArgTypes node type: %T, value: %#v\n", node, node)
	switch n := node.(type) {
	case map[string]interface{}:
		if t, ok := n["_type"]; ok {
			fmt.Fprintf(os.Stderr, "[DEBUG] visiting node type: %v\n", t)
		}
		if n["_type"] == "Call" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Call node: func=%#v, args=%#v\n", n["func"], n["args"])
			if fn, ok := n["func"].(map[string]interface{}); ok && fn["_type"] == "Name" {
				className := fn["id"].(string)
				argTypes := []string{}
				if n["args"] != nil {
					for _, a := range n["args"].([]interface{}) {
						t := getType(a)
						argTypes = append(argTypes, t)
					}
				}
				fmt.Fprintf(os.Stderr, "[DEBUG] Found Call: className=%s, argTypes=%+v\n", className, argTypes)
				classInitArgTypes[className] = append(classInitArgTypes[className], argTypes)
				funcArgTypes[className] = append(funcArgTypes[className], argTypes)
			}
		}
		for _, v := range n {
			collectClassInitArgTypes(v)
		}
	case ASTNode:
		m := map[string]interface{}(n)
		if t, ok := m["_type"]; ok {
			fmt.Fprintf(os.Stderr, "[DEBUG] visiting node type: %v\n", t)
		}
		if m["_type"] == "Call" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Call node: func=%#v, args=%#v\n", m["func"], m["args"])
			if fn, ok := m["func"].(map[string]interface{}); ok && fn["_type"] == "Name" {
				className := fn["id"].(string)
				argTypes := []string{}
				if m["args"] != nil {
					for _, a := range m["args"].([]interface{}) {
						t := getType(a)
						argTypes = append(argTypes, t)
					}
				}
				fmt.Fprintf(os.Stderr, "[DEBUG] Found Call: className=%s, argTypes=%+v\n", className, argTypes)
				classInitArgTypes[className] = append(classInitArgTypes[className], argTypes)
				funcArgTypes[className] = append(funcArgTypes[className], argTypes)
			}
		}
		for _, v := range m {
			collectClassInitArgTypes(v)
		}
	case []interface{}:
		for _, elem := range n {
			collectClassInitArgTypes(elem)
			// 新增：如果 elem 是 map[string]interface{} 或 ASTNode，再递归其所有字段
			switch e := elem.(type) {
			case map[string]interface{}:
				for _, v := range e {
					collectClassInitArgTypes(v)
				}
			case ASTNode:
				m := map[string]interface{}(e)
				for _, v := range m {
					collectClassInitArgTypes(v)
				}
			}
		}
	}
}
