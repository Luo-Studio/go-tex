use std::collections::HashMap;

use crate::error::ParseResult;
use crate::functions::{define_function_full, FunctionContext, FunctionSpec};
use crate::parse_node::ParseNode;

pub fn register(map: &mut HashMap<&'static str, FunctionSpec>) {
    define_function_full(
        map,
        &["\\html@mathml"],
        "htmlmathml",
        2, 0, None,
        false, true, true, false, false,
        handle_htmlmathml,
    );
}

fn handle_htmlmathml(
    ctx: &mut FunctionContext,
    args: Vec<ParseNode>,
    _opt_args: Vec<Option<ParseNode>>,
) -> ParseResult<ParseNode> {
    let html = ParseNode::ord_argument(args[0].clone());
    let mathml = ParseNode::ord_argument(args[1].clone());

    Ok(ParseNode::HtmlMathMl {
        mode: ctx.parser.mode,
        html,
        mathml,
        loc: None,
    })
}
