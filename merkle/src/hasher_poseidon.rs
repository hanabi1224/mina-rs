// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: Apache-2.0

use super::*;
use lockfree_object_pool::SpinLockObjectPool;
use mina_hasher::{Fp, Hashable, PoseidonHasherLegacy};
use std::marker::PhantomData;

/// Trait that provides poseidon hasher pool
/// as it's expensive to create a new hasher
pub trait PoseidonLegacyHasherPoolProvider {
    /// Item type
    type Item: Hashable;

    /// Gets hasher pool for the associated Item type
    fn get_pool<'a>() -> &'a SpinLockObjectPool<PoseidonHasherLegacy<Self::Item>>;
}

/// Macro that auto-implements PoseidonLegacyHasherPoolProvider
#[macro_export]
macro_rules! impl_poseidon_legacy_hasher_pool_provider {
    ($t:ty) => {
        impl mina_merkle::PoseidonLegacyHasherPoolProvider for $t {
            type Item = Self;

            fn get_pool<'a>() -> &'a mina_merkle::macros::lockfree_object_pool::SpinLockObjectPool<
                mina_merkle::macros::mina_hasher::PoseidonHasherLegacy<$t>,
            > {
                use mina_merkle::macros::{
                    lockfree_object_pool::SpinLockObjectPool,
                    mina_hasher::{create_legacy, Hashable, PoseidonHasherLegacy},
                    once_cell::sync::OnceCell,
                };

                static POOL: OnceCell<SpinLockObjectPool<PoseidonHasherLegacy<$t>>> =
                    OnceCell::new();
                let pool = POOL.get_or_init(|| {
                    SpinLockObjectPool::new(
                        || create_legacy(<$t as Hashable>::D::default()),
                        |_| (),
                    )
                });
                pool
            }
        }
    };
}

/// Hasher for mina binary merkle tree that uses poseidon hash
pub struct MinaPoseidonMerkleHasher<Item>
where
    Item: mina_hasher::Hashable,
{
    _pd: PhantomData<Item>,
}

impl<Item> MerkleHasher for MinaPoseidonMerkleHasher<Item>
where
    Item: mina_hasher::Hashable + PoseidonLegacyHasherPoolProvider<Item = Item>,
    <Item as mina_hasher::Hashable>::D: Default,
{
    type Item = Item;
    type Hash = Fp;
    fn hash(item: &Self::Item, _: MerkleTreeNodeMetadata) -> Self::Hash {
        use mina_hasher::Hasher;
        let pool = <Item as PoseidonLegacyHasherPoolProvider>::get_pool();
        let mut hasher = pool.pull();
        hasher.hash(item)
    }
}
