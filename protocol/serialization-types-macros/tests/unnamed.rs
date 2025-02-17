// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: Apache-2.0

#[cfg(test)]
mod tests {
    use mina_serialization_types_macros::*;

    #[derive(Debug, Clone, Eq, PartialEq, derive_more::From, derive_more::Into)]
    struct I64(pub i64);

    #[derive(Debug, Clone, Eq, PartialEq, AutoFrom)]
    #[auto_from(Bar)]
    struct Foo(i64, pub i64, i64, Vec<i64>, Option<i64>, Box<i64>);

    #[derive(Debug, Clone, Eq, PartialEq)]
    struct Bar(pub I64, I64, I64, Vec<I64>, Option<I64>, Box<I64>);

    type BarV1 = ::versioned::Versioned<Bar, 1>;

    #[test]
    fn struct_with_unnamed_fields_roundtrip() {
        let foo = Foo(3, 4, 5, vec![6, 7], Some(8), Box::new(9));

        let bar: Bar = foo.clone().into();
        let foo_from_bar: Foo = bar.into();

        assert_eq!(foo, foo_from_bar);
    }

    #[test]
    fn struct_with_unnamed_fields_roundtrip_versioned() {
        let foo = Foo(3, 4, 5, vec![6, 7], Some(8), Box::new(9));

        let bar: BarV1 = foo.clone().into();
        let foo_from_bar: Foo = bar.into();

        assert_eq!(foo, foo_from_bar);
    }
}
